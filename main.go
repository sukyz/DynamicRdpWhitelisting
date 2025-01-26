package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"os/exec"
	"strings"
	"sync"
	"time"
)

type Config struct {
	Password  string `json:"password"`
	Port      int    `json:"port"`
	RDPPort   int    `json:"rdp_port"`
}

type IPEntry struct {
	IP        string
	ExpiresAt time.Time
}

var (
	config     Config
	ipList     = make(map[string]IPEntry)
	ipListLock sync.RWMutex
)

func loadConfig() error {
	data, err := ioutil.ReadFile("config.json")
	if err != nil {
		return err
	}
	return json.Unmarshal(data, &config)
}

func addIPToFirewall(ip string) error {
	cmd := exec.Command("netsh", "advfirewall", "firewall", "add", "rule",
		"name="+ip,
		"dir=in",
		"action=allow",
		"protocol=TCP",
		"localport="+fmt.Sprintf("%d", config.RDPPort),
		"remoteip="+ip)
	return cmd.Run()
}

func removeIPFromFirewall(ip string) error {
	cmd := exec.Command("netsh", "advfirewall", "firewall", "delete", "rule",
		"name="+ip)
	return cmd.Run()
}

func cleanupExpiredIPs() {
	for {
		time.Sleep(5 * time.Minute)
		ipListLock.Lock()
		now := time.Now()
		for ip, entry := range ipList {
			if now.After(entry.ExpiresAt) {
				removeIPFromFirewall(ip)
				delete(ipList, ip)
			}
		}
		ipListLock.Unlock()
	}
}

func handleAddIP(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.ServeFile(w, r, "form.html")
		return
	}

	password := r.FormValue("password")
	if password != config.Password {
		http.Error(w, "密码错误", http.StatusUnauthorized)
		return
	}

	ip := r.RemoteAddr
	if colon := lastIndex(ip, ":"); colon != -1 {
		ip = ip[:colon]
	}

	ipListLock.Lock()
	defer ipListLock.Unlock()

	if err := addIPToFirewall(ip); err != nil {
		http.Error(w, "添加防火墙规则失败", http.StatusInternalServerError)
		return
	}

	ipList[ip] = IPEntry{
		IP:        ip,
		ExpiresAt: time.Now().Add(5 * time.Hour),
	}

	fmt.Fprintf(w, "IP %s 已成功添加到白名单，将在5小时后过期", ip)
}

func main() {
	if err := loadConfig(); err != nil {
		log.Fatal("加载配置文件失败:", err)
	}

	go cleanupExpiredIPs()

	http.HandleFunc("/", handleAddIP)
	
	log.Printf("服务启动在端口 %d\n", config.Port)
	if err := http.ListenAndServe(fmt.Sprintf(":%d", config.Port), nil); err != nil {
		log.Fatal(err)
	}
}

// 创建一个简单的HTML表单
const formHTML = `
<!DOCTYPE html>
<html>
<head>
    <title>RDP白名单管理</title>
    <meta charset="utf-8">
</head>
<body>
    <h2>添加IP到RDP白名单</h2>
    <form method="POST">
        <label for="password">密码:</label><br>
        <input type="password" id="password" name="password"><br>
        <input type="submit" value="添加当前IP">
    </form>
</body>
</html>
`

func init() {
	// 创建form.html文件
	if err := ioutil.WriteFile("form.html", []byte(formHTML), 0644); err != nil {
		log.Fatal("创建form.html失败:", err)
	}

	// 如果配置文件不存在，创建默认配置
	if _, err := os.Stat("config.json"); os.IsNotExist(err) {
		defaultConfig := Config{
			Password: "your-password-here",
			Port:     8080,
			RDPPort:  3389,
		}
		configData, _ := json.MarshalIndent(defaultConfig, "", "    ")
		if err := ioutil.WriteFile("config.json", configData, 0644); err != nil {
			log.Fatal("创建配置文件失败:", err)
		}
	}
}

func lastIndex(s, substr string) int {
	return strings.LastIndex(s, substr)
}


