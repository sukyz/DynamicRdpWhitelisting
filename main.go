package main

import (
	"embed"
	"log"
	"net"
	"os/exec"
	"time"

	"github.com/gin-gonic/gin"
)

//go:embed templates/index.html
var templateFS embed.FS

const (
	PASSWORD = "123321"
	WHITELIST_DURATION = 5 * time.Hour
)

type WhitelistManager struct {
	whitelist map[string]time.Time
}

func NewWhitelistManager() *WhitelistManager {
	return &WhitelistManager{
		whitelist: make(map[string]time.Time),
	}
}

func (wm *WhitelistManager) AddToRDPWhitelist(ip string) error {
	cmd := exec.Command("netsh", "advfirewall", "firewall", "add", "rule", 
		"name=RDP_Whitelist", 
		"dir=in", 
		"action=allow", 
		"protocol=TCP", 
		"localport=3389", 
		"remoteip=" + ip)
	
	err := cmd.Run()
	if err != nil {
		return err
	}

	wm.whitelist[ip] = time.Now().Add(WHITELIST_DURATION)
	return nil
}

func (wm *WhitelistManager) CleanupExpiredIPs() {
	for {
		time.Sleep(15 * time.Minute)
		now := time.Now()
		
		for ip, expireTime := range wm.whitelist {
			if now.After(expireTime) {
				exec.Command("netsh", "advfirewall", "firewall", "delete", "rule", 
					"name=RDP_Whitelist", 
					"remoteip=" + ip).Run()
				
				delete(wm.whitelist, ip)
			}
		}
	}
}

func main() {
	whitelistManager := NewWhitelistManager()
	
	go whitelistManager.CleanupExpiredIPs()

	r := gin.Default()

	// 使用嵌入的模板文件
	templ := template.Must(template.ParseFS(templateFS, "templates/index.html"))
	r.SetHTMLTemplate(templ)

	r.GET("/", func(c *gin.Context) {
		c.HTML(200, "index.html", nil)
	})

	r.POST("/whitelist", func(c *gin.Context) {
		password := c.PostForm("password")
		
		if password != PASSWORD {
			c.JSON(403, gin.H{"error": "密码错误"})
			return
		}

		clientIP := c.ClientIP()
		err := whitelistManager.AddToRDPWhitelist(clientIP)
		
		if err != nil {
			c.JSON(500, gin.H{"error": "添加白名单失败"})
			return
		}

		c.JSON(200, gin.H{
			"message": "IP已成功添加到RDP白名单",
			"ip": clientIP,
			"expires": time.Now().Add(WHITELIST_DURATION),
		})
	})

	log.Println("服务器启动，访问 http://localhost:8080")
	r.Run(":8080")
}
