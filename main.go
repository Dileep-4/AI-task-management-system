package main

import (
    "github.com/gin-gonic/gin"
    "github.com/golang-jwt/jwt/v4"
    "gorm.io/driver/postgres"
    "gorm.io/gorm"
    "log"
    "net/http"
    "os"
    "time"
    "github.com/gorilla/websocket"
    "encoding/json"
    "bytes"
    "io/ioutil"
    "net/http"
)

var db *gorm.DB
var clients = make(map[*websocket.Conn]bool)
var broadcast = make(chan Task)
var openAIAPIKey = "your_openai_api_key"

func init() {
    dsn := "host=localhost user=postgres password=yourpassword dbname=tasks port=5432 sslmode=disable"
    var err error
    db, err = gorm.Open(postgres.Open(dsn), &gorm.Config{})
    if err != nil {
        log.Fatal("Failed to connect to database")
    }
    db.AutoMigrate(&User{}, &Task{})
}

type User struct {
    ID       uint   `json:"id" gorm:"primaryKey"`
    Email    string `json:"email" gorm:"unique"`
    Password string `json:"-"`
}

type Task struct {
    ID          uint   `json:"id" gorm:"primaryKey"`
    Title       string `json:"title"`
    Description string `json:"description"`
    Status      string `json:"status"`
    UserID      uint   `json:"user_id"`
}

var jwtKey = []byte("secret")

type Claims struct {
    Email string `json:"email"`
    jwt.StandardClaims
}

func Login(c *gin.Context) {
    var user User
    if err := c.ShouldBindJSON(&user); err != nil {
        c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request"})
        return
    }
    token := jwt.NewWithClaims(jwt.SigningMethodHS256, &Claims{
        Email: user.Email,
        StandardClaims: jwt.StandardClaims{
            ExpiresAt: time.Now().Add(time.Hour * 24).Unix(),
        },
    })
    tokenString, _ := token.SignedString(jwtKey)
    c.JSON(http.StatusOK, gin.H{"token": tokenString})
}

func CreateTask(c *gin.Context) {
    var task Task
    if err := c.ShouldBindJSON(&task); err != nil {
        c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request"})
        return
    }
    db.Create(&task)
    broadcast <- task
    c.JSON(http.StatusOK, task)
}

func GetTasks(c *gin.Context) {
    var tasks []Task
    db.Find(&tasks)
    c.JSON(http.StatusOK, tasks)
}

func WebSocketHandler(c *gin.Context) {
    conn, err := websocket.Upgrade(c.Writer, c.Request, nil, 1024, 1024)
    if err != nil {
        log.Println("Failed to upgrade websocket:", err)
        return
    }
    defer conn.Close()
    clients[conn] = true
    for {
        task := <-broadcast
        message, _ := json.Marshal(task)
        for client := range clients {
            client.WriteMessage(websocket.TextMessage, message)
        }
    }
}

func GenerateTaskSuggestion(c *gin.Context) {
    var request struct {
        Prompt string `json:"prompt"`
    }
    if err := c.ShouldBindJSON(&request); err != nil {
        c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request"})
        return
    }

    requestBody, _ := json.Marshal(map[string]interface{}{
        "model": "gpt-3.5-turbo",
        "messages": []map[string]string{
            {"role": "user", "content": request.Prompt},
        },
    })

    req, _ := http.NewRequest("POST", "https://api.openai.com/v1/chat/completions", bytes.NewBuffer(requestBody))
    req.Header.Set("Content-Type", "application/json")
    req.Header.Set("Authorization", "Bearer "+openAIAPIKey)

    client := &http.Client{}
    resp, err := client.Do(req)
    if err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to communicate with AI API"})
        return
    }
    defer resp.Body.Close()
    body, _ := ioutil.ReadAll(resp.Body)
    c.JSON(http.StatusOK, gin.H{"suggestion": string(body)})
}

func main() {
    r := gin.Default()
    r.POST("/login", Login)
    r.POST("/tasks", CreateTask)
    r.GET("/tasks", GetTasks)
    r.GET("/ws", WebSocketHandler)
    r.POST("/ai-suggest", GenerateTaskSuggestion)
    r.Run(":8080")
}
