package main

import (
	"net/http"
	"os/exec"
	"time"
)

const (
	password = "yourSecurePassword" // 设置密码
)

func main() {
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case "GET":
			// 提供一个HTML表单
			htmlForm := `
				
```html
<!DOCTYPE html>
				<html>
				<head>
					<title>Enter Password to Whitelist IP</title>
				</head>
				<body>
					<form action="/" method="post">
						<label for="password">Password:</label>
						<input type="password" id="password" name="password"><br><br>
						<input type="submit" value="Submit">
					</form>
				</body>
				</html>
		`
		w.Write([]byte(htmlForm))
	case "POST":
		r.ParseForm()
		clientPassword := r.FormValue("password")
		clientIP := r.RemoteAddr

		if clientPassword == password {
			// 添加防火墙规则
			cmd := exec.Command("netsh", "advfirewall", "firewall", "add", "rule", "name=\"Allow RDP\"", "dir=in", "action=allow", "protocol=TCP", "localport=3389", "remoteip="+clientIP)
			if err := cmd.Run(); err != nil {
				http.Error(w, "Failed to update firewall rules", http.StatusInternalServerError)
				return
			}

			// 设置定时器，5小时后删除规则
			time.AfterFunc(5*time.Hour, func() {
				exec.Command("netsh", "advfirewall", "firewall", "delete", "rule", "name=\"Allow RDP\"", "remoteip="+clientIP).Run()
			})

			w.Write([]byte("IP added to whitelist successfully"))
		} else {
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
		}
	default:
		w.WriteHeader(http.StatusMethodNotAllowed)
	}
})

http.ListenAndServe(":8080", nil)
