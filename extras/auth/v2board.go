package auth

import (
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"strconv"
	"sync"
	"time"

	"github.com/apernet/hysteria/core/server"
)

type V2boardApiProvider struct {
	Client *http.Client
	URL    string
	userMap sync.Map  // 使用sync.Map替代全局变量和互斥锁
}

type User struct {
	ID         int     `json:"id"`
	UUID       string  `json:"uuid"`
	SpeedLimit *uint32 `json:"speed_limit"`
}

type ResponseData struct {
	Users []User `json:"users"`
}

// getUserList - 封装错误处理和日志记录
func getUserList(client *http.Client, url string) ([]User, error) {
	resp, err := client.Get(url)
	if err != nil {
		return nil, fmt.Errorf("error fetching user list: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	var responseData ResponseData
	err = json.NewDecoder(resp.Body).Decode(&responseData)
	if err != nil {
		return nil, fmt.Errorf("error decoding JSON: %v", err)
	}

	return responseData.Users, nil
}

// UpdateUsers - 动态调整更新间隔
func (v *V2boardApiProvider) UpdateUsers(interval time.Duration) {
	fmt.Println("用户列表自动更新服务已激活")
	for range time.Tick(interval) {
		userList, err := getUserList(v.Client, v.URL)
		if err != nil {
			fmt.Println("Error updating users:", err)
			continue
		}
		for _, user := range userList {
			v.userMap.Store(user.UUID, user)
		}
	}
}

// Authenticate - 使用sync.Map进行线程安全的读写
func (v *V2boardApiProvider) Authenticate(addr net.Addr, auth string, tx uint64) (bool, string) {
	if user, ok := v.userMap.Load(auth); ok {
		return true, strconv.Itoa(user.(User).ID)
	}
	return false, ""
}
