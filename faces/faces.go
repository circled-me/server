package faces

import (
	"io"
	"net/http"
	"os/exec"
	"sync"
	"time"

	"log"

	"github.com/gin-gonic/gin"
)

var (
	pipesReady    chan struct{}  = make(chan struct{})
	buf           []byte         = make([]byte, 256*1024)
	scriptRunning                = false
	stdin         io.WriteCloser = nil
	stdout        io.ReadCloser  = nil
	mutex                        = sync.Mutex{}
	lastUsed                     = time.Now()
)

func init() {
	go backgroundChecker()
}

func shutdown() {
	scriptRunning = false
	stdin.Close()
	stdout.Close()
	stdin = nil
	stdout = nil
	log.Println("Python script stopped")
}

func backgroundChecker() {
	for {
		mutex.Lock()
		if scriptRunning {
			if time.Since(lastUsed) > 20*time.Second && stdin != nil && stdout != nil {
				shutdown()
			} else {
				if writeAndRead("ping") != "pong" {
					shutdown()
				}
			}
		}
		mutex.Unlock()
		time.Sleep(10 * time.Second)
	}
}

func writeAndRead(line string) string {
	_, err := stdin.Write([]byte(line + "\n"))
	if err != nil {
		log.Printf("Error writing to Python script: %v", err)
		shutdown()
		return ""
	}
	// Read the result from the Python script
	n, err := stdout.Read(buf)
	if err != nil {
		log.Printf("Error reading from Python script: %v", err)
		shutdown()
		return ""
	}
	if n == 0 {
		log.Println("Python script returned empty result")
		return ""
	}
	// Strip the trailing newline
	return string(buf[0 : n-1])
}

func Detect(imgPath string) (FaceDetectionResult, error) {
	log.Printf("Detecting faces in %s", imgPath)
	mutex.Lock()
	log.Printf("Got lock for %s", imgPath)

	lastUsed = time.Now()

	// If the Python script is not running, start it
	if !scriptRunning {
		log.Println("Starting Python script...")
		scriptRunning = true
		go runPythonScript()
		// Wait until the in/out pipes are ready
		<-pipesReady
	} else {
		log.Println("Python script already running")
	}
	result := writeAndRead(imgPath)
	log.Println("Done detecting faces")
	mutex.Unlock()
	log.Println("Released lock")

	return toFacesResult([]byte(result))
}

func runPythonScript() {
	// Start a new Python sub process and intercept its input/output
	cmd := exec.Command("python3", "./faces/face-extract.py")
	stdin, _ = cmd.StdinPipe()
	stdout, _ = cmd.StdoutPipe()
	stderr, _ := cmd.StderrPipe()
	err := cmd.Start()
	if err != nil {
		log.Printf("Error running Python script: %v", err)
	}
	// Notify the main goroutine that the pipes are ready
	pipesReady <- struct{}{}

	err = cmd.Wait()
	if err != nil {
		// This below is not working
		n, _ := stderr.Read(buf)
		log.Printf("Error running Python script: %v, output: %s", err, string(buf[:n]))
	}
}

func DetectHandler(c *gin.Context) {
	imgPath := c.Query("img")
	if imgPath == "" {
		c.String(http.StatusBadRequest, "img parameter missing")
		return
	}
	result, err := Detect(imgPath)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, result)
}
