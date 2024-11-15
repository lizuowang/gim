package http

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/lizuowang/gim/service/sys"
)

func HomePage(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintf(w, "Home Page")
}

func GetSysInfo(w http.ResponseWriter, r *http.Request) {
	sysInfo := sys.GetSysCollectInfo()
	bytes, err := json.Marshal(sysInfo)
	if err != nil {
		fmt.Fprintf(w, "获取系统信息失败")
	}
	fmt.Fprintf(w, string(bytes))
}
