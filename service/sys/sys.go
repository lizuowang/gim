package sys

import (
	"runtime"

	"github.com/shirou/gopsutil/v3/cpu"
	"github.com/shirou/gopsutil/v3/mem"

	"github.com/lizuowang/gim/pkg/types"
	rpcClient "github.com/lizuowang/gim/service/api/client"
	"github.com/lizuowang/gim/service/ws"
)

// 获取本机系统信息
func GetSysInfo() (managerInfo *types.SysInfo) {
	managerInfo = &types.SysInfo{}
	managerInfo.NumGoroutine = runtime.NumGoroutine()
	managerInfo.NumCPU = runtime.NumCPU()
	managerInfo.ManagerInfo = ws.GetManagerInfo()
	managerInfo.Version = ws.Config.Version

	// 获取 CPU 使用率
	cpuPercent, err := cpu.Percent(0, false)
	if err != nil {
		managerInfo.CPUPercent = 0
	} else {
		managerInfo.CPUPercent = cpuPercent[0]
	}

	// 获取内存使用率
	vmStat, err := mem.VirtualMemory()
	if err != nil {
		managerInfo.MemPercent = 0
	} else {
		managerInfo.MemPercent = vmStat.UsedPercent
	}

	managerInfo.SubMsgList = ws.GetSubMsgList()
	managerInfo.Name = ws.ServerName
	return
}

// 获取所有节点系统信息
func getAllNodeSysInfo() (allSysInfo []*types.SysInfo, err error) {
	if ws.Config.RunMode == ws.RunModeLocal { //单机模式
		sysInfo := GetSysInfo()
		allSysInfo = append(allSysInfo, sysInfo)
	} else {
		allSysInfo, err = rpcClient.GetAllSysInfo()
	}
	return
}

// 获取系统统计信息
func GetSysCollectInfo() (sysCollectInfo *types.SycCollectInfo) {
	sysCollectInfo = types.NewSycCollectInfo()
	allSysInfo, err := getAllNodeSysInfo()
	if err != nil {
		return
	}

	sysCollectInfo.NodeList = allSysInfo
	for _, sysInfo := range allSysInfo {
		sysCollectInfo.NodeNum++
		sysCollectInfo.Version = sysInfo.Version
		sysCollectInfo.NumCPU += sysInfo.NumCPU
		sysCollectInfo.NumGoroutine += sysInfo.NumGoroutine

		// 客户端数
		sysCollectInfo.ManagerInfo.ClientsLen += sysInfo.ManagerInfo.ClientsLen
		sysCollectInfo.ManagerInfo.TGroupLen += sysInfo.ManagerInfo.TGroupLen
	}

	return
}
