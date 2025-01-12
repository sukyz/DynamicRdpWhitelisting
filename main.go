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
