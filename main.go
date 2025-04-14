package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"regexp"
	"time"

	"github.com/cloudflare/cloudflare-go"
)

type Config struct {
	APIToken    string `json:"apiToken"`
	ZoneID      string `json:"zoneID"`
	RecordName  string `json:"recordName"`
	CheckPeriod int    `json:"checkPeriod"`
	IPAPI       string `json:"ipAPI"`
}

func loadConfig(path string) (*Config, time.Duration, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, 0, fmt.Errorf("打开配置文件失败: %v", err)
	}
	defer file.Close()

	var config Config
	if err := json.NewDecoder(file).Decode(&config); err != nil {
		return nil, 0, fmt.Errorf("解析配置文件失败: %v", err)
	}

	duration := time.Duration(config.CheckPeriod) * time.Second

	return &config, duration, nil
}

func getPublicIPByIPIPNET() (string, error) {
	var ipRegex = regexp.MustCompile(`当前 IP：(\d+\.\d+\.\d+\.\d+)`)

	ipAPI := "https://myip.ipip.net"

	resp, err := http.Get(ipAPI)
	if err != nil {
		return "", fmt.Errorf("获取IP失败: %v", err)
	}
	defer resp.Body.Close()

	buf := make([]byte, 1024)
	n, _ := resp.Body.Read(buf)
	ipMatch := ipRegex.FindStringSubmatch(string(buf[:n]))

	if len(ipMatch) < 2 {
		return "", fmt.Errorf("IP解析失败")
	}
	return ipMatch[1], nil
}

func updateDNS(api *cloudflare.API, zoneID, recordName, currentIP string) error {
	ctx := context.Background()

	records, _, err := api.ListDNSRecords(ctx, cloudflare.ZoneIdentifier(zoneID), cloudflare.ListDNSRecordsParams{
		Name: recordName,
	})
	if err != nil {
		return fmt.Errorf("获取记录失败: %v", err)
	}

	if len(records) == 0 {
		_, err = api.CreateDNSRecord(ctx, cloudflare.ZoneIdentifier(zoneID), cloudflare.CreateDNSRecordParams{
			Type:    "A",
			Name:    recordName,
			Content: currentIP,
			TTL:     60,
			Proxied: cloudflare.BoolPtr(false),
		})
	} else {
		_, err = api.UpdateDNSRecord(ctx, cloudflare.ZoneIdentifier(zoneID), cloudflare.UpdateDNSRecordParams{
			ID:      records[0].ID,
			Type:    "A",
			Name:    recordName,
			Content: currentIP,
		})
	}

	if err != nil {
		return fmt.Errorf("更新记录失败: %v", err)
	}
	return nil
}

func mainLoop(config *Config, api *cloudflare.API) {
	var currentIP string
	var err error

	if config.IPAPI == "ipip.net" {
		currentIP, err = getPublicIPByIPIPNET()
	} else {
		panic("不合法的 IP 地址 API")
	}

	if err != nil {
		fmt.Printf("[%s] 错误: %v\n", time.Now().Format("2006-01-02 15:04:05"), err)
	} else {
		fmt.Printf("[%s] 当前公网IP: %s\n", time.Now().Format("2006-01-02 15:04:05"), currentIP)

		if err := updateDNS(api, config.ZoneID, config.RecordName, currentIP); err != nil {
			fmt.Printf("[%s] 更新失败: %v\n", time.Now().Format("2006-01-02 15:04:05"), err)
		} else {
			fmt.Printf("[%s] 记录更新成功\n", time.Now().Format("2006-01-02 15:04:05"))
		}
	}
}

func main() {
	config, checkPeriod, err := loadConfig("config.json")
	if err != nil {
		panic(fmt.Sprintf("配置加载失败: %v", err))
	}

	api, err := cloudflare.NewWithAPIToken(config.APIToken)
	if err != nil {
		panic(fmt.Sprintf("API初始化失败: %v", err))
	}

	ticker := time.NewTicker(checkPeriod)
	defer ticker.Stop()

	fmt.Printf("[%s] DDNS服务已启动，检查间隔：%v\n",
		time.Now().Format("2006-01-02 15:04:05"), checkPeriod)

	mainLoop(config, api)

	for range ticker.C {
		mainLoop(config, api)
	}
}
