package main

import (
	"bytes"
	"crypto/rand"
	"encoding/csv"
	"flag"
	"fmt"
	"log"
	mathRand "math/rand"
	"os"
	"time"

	"github.com/cykyes/flupoc-go/client"
)

// 使用示例:
// go run .\cmd\echo_client\main.go --addr="127.0.0.1:5128" --cert="path/to/cert.crt" --ca="path/to/ca.crt"

const (
	minDataSize = 0                // 最小数据大小 0 字节
	maxDataSize = 20 * 1024 * 1024 // 最大数据大小 20MB
	minInterval = 0                // 最小间隔 0 秒
	maxInterval = 60               // 最大间隔 60 秒
	csvFileName = "echo_test_log.csv"
)

func main() {
	addr := flag.String("addr", "127.0.0.1:5128", "服务器地址")
	certFile := flag.String("cert", "", "客户端证书文件路径 (可选，用于 mTLS)")
	keyFile := flag.String("key", "", "客户端私钥文件路径 (可选，用于 mTLS)")
	caFile := flag.String("ca", "", "CA 证书文件路径 (用于验证服务器)")
	insecure := flag.Bool("insecure", false, "跳过服务器证书验证")
	runs := flag.Int("runs", 0, "运行次数 (0 表示无限循环)")
	flag.Parse()

	if *caFile == "" && !*insecure {
		log.Fatal("必须提供 --ca 参数或使用 --insecure 跳过证书验证")
	}

	// 初始化随机数生成器
	mathRand.Seed(time.Now().UnixNano())

	// 创建客户端
	opts := client.Options{
		CertFile:     *certFile,
		KeyFile:      *keyFile,
		CAFile:       *caFile,
		Insecure:     *insecure,
		DialTimeout:  10 * time.Second,
		ReadTimeout:  30 * time.Second,
		WriteTimeout: 30 * time.Second,
	}

	c, err := client.New(opts)
	if err != nil {
		log.Fatalf("创建客户端失败: %v", err)
	}

	// 初始化 CSV 文件
	if err := initCSVFile(csvFileName); err != nil {
		log.Fatalf("初始化 CSV 文件失败: %v", err)
	}

	log.Printf("开始测试，目标地址: %s", *addr)
	log.Printf("CSV 日志文件: %s", csvFileName)

	runCount := 0
	for {
		// 检查是否达到运行次数限制
		if *runs > 0 {
			runCount++
			if runCount > *runs {
				log.Printf("已完成 %d 次运行，程序退出", *runs)
				break
			}
		}

		// 生成随机数据大小 (0~20MB)
		dataSize := mathRand.Intn(maxDataSize + 1)
		data := make([]byte, dataSize)
		if dataSize > 0 {
			if _, err := rand.Read(data); err != nil {
				log.Printf("生成随机数据失败: %v", err)
				continue
			}
		}

		// 记录发送时间（东八区）
		beijingLocation := time.FixedZone("CST", 8*3600)
		sendTime := time.Now().In(beijingLocation)

		log.Printf("发送请求，数据大小: %d 字节", dataSize)

		// 发送请求并计时
		startTime := time.Now()
		resp, err := c.Do(*addr, "POST", "/echo", data)
		duration := time.Since(startTime)

		// 检查响应是否正确
		var isMatch bool
		var errorMsg string
		if err != nil {
			isMatch = false
			errorMsg = err.Error()
			log.Printf("请求失败: %v", err)
		} else {
			// Body 已经是 []byte 类型，直接比较
			isMatch = bytes.Equal(data, resp.Body)
			if isMatch {
				log.Printf("响应匹配，耗时: %v", duration)
			} else {
				log.Printf("响应不匹配，发送大小: %d，接收大小: %d，耗时: %v",
					len(data), len(resp.Body), duration)
				errorMsg = fmt.Sprintf("数据不匹配，发送=%d字节，接收=%d字节", len(data), len(resp.Body))
			}
		}

		// 记录到 CSV
		if err := appendToCSV(csvFileName, CSVRecord{
			SendTime: sendTime,
			IsMatch:  isMatch,
			Duration: duration,
			DataSize: dataSize,
			ErrorMsg: errorMsg,
		}); err != nil {
			log.Printf("写入 CSV 失败: %v", err)
		}

		// 生成随机间隔 (0~60秒)
		interval := mathRand.Intn(maxInterval + 1)

		if *runs == 0 || runCount < *runs {
			log.Printf("等待 %d 秒后发送下一个请求...", interval)
			time.Sleep(time.Duration(interval) * time.Second)
		}
	}
}

// CSVRecord 表示一条 CSV 记录
type CSVRecord struct {
	SendTime time.Time
	IsMatch  bool
	Duration time.Duration
	DataSize int
	ErrorMsg string
}

// initCSVFile 初始化 CSV 文件，如果文件不存在则创建并写入表头
func initCSVFile(filename string) error {
	// 检查文件是否已存在
	if _, err := os.Stat(filename); err == nil {
		// 文件已存在，无需初始化
		return nil
	}

	// 创建文件并写入表头
	file, err := os.Create(filename)
	if err != nil {
		return fmt.Errorf("创建文件失败: %w", err)
	}
	defer file.Close()

	// 写入 UTF-8 BOM 以便 Excel 正确识别中文
	if _, err := file.Write([]byte{0xEF, 0xBB, 0xBF}); err != nil {
		return fmt.Errorf("写入 BOM 失败: %w", err)
	}

	writer := csv.NewWriter(file)
	defer writer.Flush()

	// 写入表头（数据大小单位改为 MB）
	header := []string{"发送日期时间(东八区)", "数据是否相同", "往返时间(毫秒)", "数据大小(MB)", "错误信息"}
	if err := writer.Write(header); err != nil {
		return fmt.Errorf("写入表头失败: %w", err)
	}

	return nil
}

// appendToCSV 追加一条记录到 CSV 文件
func appendToCSV(filename string, record CSVRecord) error {
	file, err := os.OpenFile(filename, os.O_APPEND|os.O_WRONLY, 0644)
	if err != nil {
		return fmt.Errorf("打开文件失败: %w", err)
	}
	defer file.Close()

	writer := csv.NewWriter(file)
	defer writer.Flush()

	// 格式化记录
	isMatchStr := "是"
	if !record.IsMatch {
		isMatchStr = "否"
	}

	row := []string{
		record.SendTime.Format("2006-01-02 15:04:05"),
		isMatchStr,
		fmt.Sprintf("%.2f", float64(record.Duration.Microseconds())/1000.0),
		fmt.Sprintf("%.2f", float64(record.DataSize)/(1024*1024)),
		record.ErrorMsg,
	}

	if err := writer.Write(row); err != nil {
		return fmt.Errorf("写入记录失败: %w", err)
	}

	return nil
}
