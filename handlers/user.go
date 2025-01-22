package handlers

import (
	"errors"
	"log"
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
	Email    string `json:"email" binding:"required"`
	Password string `json:"password" binding:"required"`
	Token    string `json:"token"`
	New      bool   `json:"new"`
}

type UserInfo struct {
	ID          uint64 `json:"id"`
	Name        string `json:"name"`
	Email       string `json:"email"`
	Permissions []int  `json:"permissions"`
	Bucket      uint64 `json:"bucket"`
	Quota       int64  `json:"quota"` // in MB
}

type UserStatusResponse struct {
	Error       string `json:"error"`
	Name        string `json:"name"`
	UserID      uint64 `json:"user_id"`
	PushToken   string `json:"push_token"`
	Permissions []int  `json:"permissions"`
	BucketUsage int64  `json:"bucket_usage"`
	BucketQuota int64  `json:"bucket_quota"`
}

type UserSaveResponse struct {
	Error string `json:"error"`
	Token string `json:"token"`
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
	if db.Instance.Where("email = ? and password = ''", postReq.Token).Find(&user).Error != nil || user.ID == 0 {
		return errors.New("invalid token")
	}
	if !isValidLogin(postReq.Email) {
		return errors.New("login cannot contain empty spaces and must start with a letter or a number")
	}
	user.Email = postReq.Email
	user.SetPassword(postReq.Password)
	err = db.Instance.Where("id = ?", user.ID).Updates(&models.User{
		Email:    user.Email,
		Password: user.Password,
		PassSalt: user.PassSalt,
	}).Error
	if err != nil {
		return errors.New("user with the same login seems to exist")
	}
	return nil
}

func createFirstUser(postReq *UserLoginRequest) (err error) {
	user, err := models.UserCreate(postReq.Email, postReq.Email, postReq.Password)
	if err != nil {
		return errors.New("DB error 2")
	}
	// Add all permissions to the first user
	for _, permission := range models.AllPermissions {
		err = db.Instance.Save(&models.Grant{
			GrantorID:  user.ID,
			UserID:     user.ID,
			Permission: permission,
		}).Error
		if err != nil {
			return errors.New("DB error 3")
		}
	}
	return nil
}

func newUserStatusResponse(user *models.User, details bool) UserStatusResponse {
	result := UserStatusResponse{
		UserID:      user.ID,
		Name:        user.Name,
		Permissions: user.GetPermissions(),
		BucketUsage: -1,
		BucketQuota: -1,
	}
	if details {
		if user.PushToken == "" {
			user.SetNewPushToken()
		}
		result.PushToken = user.PushToken
		result.BucketQuota = user.Quota
		result.BucketUsage = user.GetUsage()
		if result.BucketQuota == 0 {
			// Unlimited - return the actual storage available space (if possible)
			available, _ := user.Bucket.GetSpaceInfo()
			if available >= 0 {
				result.BucketQuota = available / 1024 / 1024
			}
		}
	}
	return result
}

func numUsers() (result int) {
	db.Instance.Raw("select count(*) from users").Scan(&result)
	return
}

func UserLogin(c *gin.Context) {
	postReq := UserLoginRequest{}
	err := c.ShouldBindJSON(&postReq)
	if err != nil {
		c.JSON(http.StatusBadRequest, Response{err.Error()})
		return
	}
	if postReq.Token != "" {
		// New user has been invited
		if err = createFromToken(&postReq); err != nil {
			c.JSON(http.StatusBadRequest, Response{err.Error()})
			return
		}
	} else if postReq.New {
		// Check if we have a brand new instance
		if numUsers() != 0 {
			c.JSON(http.StatusForbidden, NopeResponse)
			return
		}
		if err = createFirstUser(&postReq); err != nil {
			c.JSON(http.StatusBadRequest, Response{err.Error()})
			return
		}
	}
	// Proceed with standard login
	user, success := models.UserLogin(postReq.Email, postReq.Password)
	if !success {
		c.JSON(http.StatusUnauthorized, Response{"Incorrect username or password"})
		return
	}
	session := auth.LoadSession(c)
	session.Set("id", user.ID)
	_ = session.Save()

	c.JSON(http.StatusOK, newUserStatusResponse(&user, false))
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

func UserSave(c *gin.Context, adminUser *models.User) {
	req := UserInfo{}
	err := c.ShouldBindWith(&req, binding.JSON)
	if err != nil {
		c.JSON(http.StatusBadRequest, Response{err.Error()})
		return
	}
	if req.Bucket == 0 {
		c.JSON(http.StatusBadRequest, Response{"select storage bucket"})
		return
	}
	// Cleanup
	req.Name = cleanupName(req.Name)
	if req.Name == "" {
		c.JSON(http.StatusBadRequest, Response{"empty name"})
		return
	}
	token := ""
	user := models.User{ID: req.ID}
	if user.ID > 0 {
		if err = db.Instance.Preload("Grants").Find(&user).Error; err != nil {
			c.JSON(http.StatusInternalServerError, DBError1Response)
			return
		}
	} else {
		// New user with random email (login)
		// They will choose their login later
		user.Email = utils.Rand16BytesToBase62()
		token = user.Email
	}
	user.BucketID = &req.Bucket
	user.Quota = req.Quota
	user.Name = req.Name
	for _, g := range user.Grants {
		db.Instance.Delete(&g)
	}
	user.Grants = []models.Grant{}
	for _, p := range req.Permissions {
		user.Grants = append(user.Grants, models.Grant{
			GrantorID:  adminUser.ID,
			Permission: models.Permission(p),
		})
	}
	if err = db.Instance.Save(&user).Error; err != nil {
		c.JSON(http.StatusInternalServerError, DBError2Response)
		return
	}
	c.JSON(http.StatusOK, UserSaveResponse{
		Token: token,
	})
}

func UserDelete(c *gin.Context, loggedUser *models.User) {
	req := UserInfo{}
	err := c.ShouldBindWith(&req, binding.JSON)
	if err != nil {
		c.JSON(http.StatusBadRequest, Response{err.Error()})
		return
	}
	if !loggedUser.HasPermission(models.PermissionAdmin) && loggedUser.ID != req.ID {
		c.JSON(http.StatusForbidden, NopeResponse)
		return
	}
	user := models.User{ID: req.ID}
	if db.Instance.First(&user).Error != nil {
		c.JSON(http.StatusBadRequest, Response{"Invalid user"})
		return
	}
	log.Printf("Will delete user: %d", user.ID)
	user.Password = ""
	db.Instance.Save(&user)
	// TODO: Delete their personal assets later as a background task, keep the group ones
	c.JSON(http.StatusOK, OKResponse)
}

func UserReInvite(c *gin.Context, currentUser *models.User) {
	req := UserInfo{}
	err := c.ShouldBindWith(&req, binding.JSON)
	if err != nil {
		c.JSON(http.StatusBadRequest, Response{err.Error()})
		return
	}
	user := models.User{ID: req.ID}
	if user.ID <= 0 {
		c.JSON(http.StatusBadRequest, Response{"hmmmm"})
		return
	}
	if err = db.Instance.Find(&user).Error; err != nil {
		c.JSON(http.StatusInternalServerError, DBError1Response)
		return
	}
	user.Email = utils.Rand16BytesToBase62()
	user.Password = ""
	if err = db.Instance.Save(&user).Error; err != nil {
		c.JSON(http.StatusInternalServerError, DBError2Response)
		return
	}
	c.JSON(http.StatusOK, UserSaveResponse{
		Token: user.Email,
	})
}

func UserGetStatus(c *gin.Context, user *models.User) {
	c.JSON(http.StatusOK, newUserStatusResponse(user, true))
}

func UserList(c *gin.Context, user *models.User) {
	users := []models.User{}
	err := db.Instance.Preload("Grants").Order("created_at ASC").Find(&users).Error
	if err != nil {
		c.JSON(http.StatusInternalServerError, DBError1Response)
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
			Quota:       u.Quota,
			Permissions: u.GetPermissions(),
		}
		result = append(result, userInfo)
	}
	c.JSON(http.StatusOK, result)
}

func UserLogout(c *gin.Context, user *models.User) {
	session := auth.LoadSession(c)
	session.LogoutUser()
	c.Status(http.StatusNoContent)
}
