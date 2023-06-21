package handlers

import (
	"errors"
	"net/http"
	"server/auth"
	"server/db"
	"server/models"
	"server/utils"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/gin-gonic/gin/binding"
)

type UserLoginRequest struct {
	Email    string `form:"email" binding:"required"`
	Password string `form:"password" binding:"required"`
	Token    string `form:"token"`
	New      bool   `form:"new"`
}

type UserInfo struct {
	ID          uint64 `json:"id"`
	Name        string `json:"name"`
	Email       string `json:"email"`
	Permissions []int  `json:"permissions"`
	Bucket      uint64 `json:"bucket"`
}

func isValidLogin(l string) bool {
	return !strings.ContainsAny(l, " \t\n\r") &&
		len(l) > 0 &&
		((l[0] >= 'a' && l[0] <= 'z') ||
			(l[0] >= 'A' && l[0] <= 'Z') ||
			(l[0] >= '0' && l[0] <= '9'))
}

func createFromToken(postReq *UserLoginRequest) (err error) {
	user := models.User{}
	if db.Instance.Where("email = ?", postReq.Token).Find(&user).Error != nil || user.ID == 0 {
		return errors.New("Invalid token")
	}
	if !isValidLogin(postReq.Email) {
		return errors.New("Login cannot contain empty spaces and must start with a letter or a number")
	}
	user.Email = postReq.Email
	user.SetPassword(postReq.Password)
	err = db.Instance.Where("id = ?", user.ID).Updates(&models.User{
		Email:    user.Email,
		Password: user.Password,
		PassSalt: user.PassSalt,
	}).Error
	if err != nil {
		return errors.New("User with the same login seems to exist")
	}
	return nil
}

func createFirstUser(postReq *UserLoginRequest) (err error) {
	user, err := models.UserCreate(postReq.Email, postReq.Email, postReq.Password)
	if err != nil {
		return errors.New("DB error 2")
	}
	err = db.Instance.Save(&models.Grant{
		GrantorID:  user.ID,
		UserID:     user.ID,
		Permission: models.PermissionAdmin,
	}).Error
	if err != nil {
		return errors.New("DB error 3")
	}
	return nil
}

func UserLogin(c *gin.Context) {
	postReq := UserLoginRequest{}
	err := c.ShouldBindWith(&postReq, binding.Form)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if postReq.Token != "" {
		// New user has been invited
		if err = createFromToken(&postReq); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
	} else if postReq.New {
		// Check if we have a brand new instance
		if err = createFirstUser(&postReq); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
	}
	// Proceed with standard login
	user, err := models.UserLogin(postReq.Email, postReq.Password)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": err.Error()})
		return
	}
	permissions := user.GetPermissions()
	session := auth.LoadSession(c)
	session.Set("id", user.ID)
	session.Set("permissions", permissions)
	session.Save()
	c.JSON(http.StatusOK, gin.H{
		"error":       "",
		"name":        user.Email,
		"user_id":     user.ID,
		"permissions": permissions,
	})
}

func cleanupName(name string) string {
	name = strings.Trim(name, " \n\r")
	for strings.Contains(name, "  ") {
		name = strings.ReplaceAll(name, "  ", " ")
	}
	if len(name) > 50 {
		name = name[:50]
	}
	return name
}

func UserSave(c *gin.Context) {
	session := auth.LoadSession(c)
	currentUser := session.User()
	if currentUser.ID == 0 || !currentUser.HasPermission(models.PermissionAdmin) {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "access denied"})
		return
	}
	req := UserInfo{}
	err := c.ShouldBindWith(&req, binding.JSON)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if req.Bucket == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "select storage bucket"})
		return
	}
	// Cleanup
	req.Name = cleanupName(req.Name)
	if req.Name == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "empty name"})
		return
	}
	token := ""
	user := models.User{ID: req.ID}
	if user.ID > 0 {
		if err = db.Instance.Preload("Grants").Find(&user).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
	} else {
		// New user with random email (login)
		// They can choose their login later
		user.Email = utils.Rand16BytesToBase62()
		token = user.Email
	}
	user.BucketID = &req.Bucket
	user.Name = req.Name
	for _, g := range user.Grants {
		db.Instance.Delete(&g)
	}
	user.Grants = []models.Grant{}
	for _, p := range req.Permissions {
		user.Grants = append(user.Grants, models.Grant{
			GrantorID:  currentUser.ID,
			Permission: models.Permission(p),
		})
	}
	if err = db.Instance.Save(&user).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"error": "",
		"token": token,
	})
}

func UserGetPermissions(c *gin.Context) {
	session := auth.LoadSession(c)
	user := session.User()
	if user.ID == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "Not Found", "name": "", "permissions": []int{}})
		return
	}
	c.JSON(http.StatusOK, gin.H{"error": "", "name": user.Name, "permissions": user.GetPermissions()})
}

func UserList(c *gin.Context) {
	session := auth.LoadSession(c)
	user := session.User()
	if user.ID == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "Not Found", "name": "", "permissions": []int{}})
		return
	}
	users := []models.User{}
	err := db.Instance.Preload("Grants").Order("created_at ASC").Find(&users).Error
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "DB error 1"})
		return
	}
	result := []UserInfo{}
	for _, u := range users {
		bucket := uint64(0)
		if u.BucketID != nil {
			bucket = *u.BucketID
		}
		userInfo := UserInfo{
			ID:          u.ID,
			Name:        u.Name,
			Email:       u.Email,
			Bucket:      bucket,
			Permissions: u.GetPermissions(),
		}
		result = append(result, userInfo)
	}
	c.JSON(http.StatusOK, result)
}
