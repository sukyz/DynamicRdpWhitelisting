package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net"
	"net/http"
	"os/exec"
	"sync"
	"time"

	"github.com/gorilla/mux"
)

type WhitelistManager struct {
	mu        sync.Mutex
	whitelist map[string]time.Time
	password  string
}

func NewWhitelistManager(password string) *WhitelistManager {
	wm := &WhitelistManager{
		whitelist: make(map[string]time.Time),
		password:  password,
	}
	
	// 定期清理过期IP
	go wm.cleanupExpiredIPs()
	
	return wm
}

func (wm *WhitelistManager) cleanupExpiredIPs() {
	for {
		time.Sleep(5 * time.Minute)
		wm.mu.Lock()
		for ip, addTime := range wm.whitelist {
			if time.Since(addTime) > 5*time.Hour {
				wm.removeIPFromFirewall(ip)
				delete(wm.whitelist, ip)
			}
		}
		wm.mu.Unlock()
	}
}

func (wm *WhitelistManager) addIPToFirewall(ip string) error {
	cmd := exec.Command("powershell", "-Command", 
		fmt.Sprintf("New-NetFirewallRule -Name 'RDP_Dynamic_Whitelist_%s' -DisplayName 'Dynamic RDP Access' -Direction Inbound -Protocol TCP -LocalPort 3389 -Action Allow -RemoteAddress %s", ip, ip))
	return cmd.Run()
}

func (wm *WhitelistManager) removeIPFromFirewall(ip string) error {
	cmd := exec.Command("powershell", "-Command", 
		fmt.Sprintf("Remove-NetFirewallRule -Name 'RDP_Dynamic_Whitelist_%s'", ip))
	return cmd.Run()
}

func (wm *WhitelistManager) AuthorizeRDP(w http.ResponseWriter, r *http.Request) {
	var request struct {
		Password string `json:"password"`
		IP       string `json:"ip"`
	}

	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		http.Error(w, "Invalid request", http.StatusBadRequest)
		return
	}

	// 验证密码
	if request.Password != wm.password {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	// 验证IP地址
	if net.ParseIP(request.IP) == nil {
		http.Error(w, "Invalid IP address", http.StatusBadRequest)
		return
	}

	wm.mu.Lock()
	defer wm.mu.Unlock()

	// 添加到防火墙
	if err := wm.addIPToFirewall(request.IP); err != nil {
		http.Error(w, "Failed to add IP to firewall", http.StatusInternalServerError)
		return
	}

	// 记录白名单
	wm.whitelist[request.IP] = time.Now()

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{
		"status":  "success",
		"message": fmt.Sprintf("IP %s authorized for 5 hours", request.IP),
	})
}

func main() {
	// 设置管理密码
	password := "your_secure_password_here"
	
	wm := NewWhitelistManager(password)

	r := mux.NewRouter()
	r.HandleFunc("/authorize", wm.AuthorizeRDP).Methods("POST")

	// 启动Web服务
	port := 8062
	log.Printf("服务启动，监听端口 %d", port)
	log.Fatal(http.ListenAndServe(fmt.Sprintf(":%d", port), r))
}
