package pcscommand

import (
	"bytes"
	"encoding/json"
	"fmt"
	"image"
	_ "image/gif"
	_ "image/jpeg"
	_ "image/png"
	"io"
	"net/http"
	"net/http/cookiejar"
	"os"
	"regexp"
	"strings"
	"time"

	"github.com/makiuchi-d/gozxing"
	"github.com/makiuchi-d/gozxing/qrcode"
	"github.com/mdp/qrterminal/v3"
	"github.com/qjfoidnh/BaiduPCS-Go/internal/pcsconfig"
	"github.com/qjfoidnh/BaiduPCS-Go/requester"
)

const (
	// QRCodeGetURL 获取二维码接口
	QRCodeGetURL = "https://passport.baidu.com/v2/api/getqrcode?lp=pc"
	// QRCodeChannelURL 轮询扫码状态接口
	QRCodeChannelURL = "https://passport.baidu.com/channel/unicast?channel_id=%s&callback="
	// QRCodeLoginURL 使用临时BDUSS完成登录接口
	QRCodeLoginURL = "https://passport.baidu.com/v3/login/main/qrbdusslogin?bduss=%s"
	// DefaultQRCodeTimeout 默认二维码超时时间
	DefaultQRCodeTimeout = 5 * time.Minute
	// PollInterval 轮询间隔
	PollInterval = 2 * time.Second
)

// QRCodeStatus 扫码状态
type QRCodeStatus int

const (
	QRStatusWaiting   QRCodeStatus = iota // 等待扫码
	QRStatusScanned                       // 已扫码，等待确认
	QRStatusConfirmed                     // 已确认
	QRStatusExpired                       // 已过期
	QRStatusError                         // 错误
)

func (s QRCodeStatus) String() string {
	switch s {
	case QRStatusWaiting:
		return "waiting"
	case QRStatusScanned:
		return "scanned"
	case QRStatusConfirmed:
		return "confirmed"
	case QRStatusExpired:
		return "expired"
	default:
		return "error"
	}
}

// QRCodeInfo 二维码信息
type QRCodeInfo struct {
	Sign     string    `json:"sign"`      // 二维码标识符
	ImgURL   string    `json:"img_url"`   // 二维码图片URL
	ASCIIQA  string    `json:"ascii_qr"`  // ASCII格式二维码
	ExpireAt time.Time `json:"expire_at"` // 过期时间
}

// QRCodeResponse 获取二维码API响应
type QRCodeResponse struct {
	ImgURL string `json:"imgurl"`
	Sign   string `json:"sign"`
	Errno  int    `json:"errno"`
}

// ChannelResponse 轮询状态API响应
type ChannelResponse struct {
	Errno    int    `json:"errno"`
	ChannelV string `json:"channel_v"`
}

// ChannelData 轮询状态数据
type ChannelData struct {
	Status int    `json:"status"`
	V      string `json:"v"` // 临时BDUSS
}

// QRLoginResult 扫码登录结果
type QRLoginResult struct {
	Status    QRCodeStatus `json:"status"`
	TempBDUSS string       `json:"temp_bduss,omitempty"`
	Message   string       `json:"message"`
}

// QRLoginClient 扫码登录客户端
type QRLoginClient struct {
	httpClient *http.Client
	userAgent  string
}

// NewQRLoginClient 创建扫码登录客户端
func NewQRLoginClient() *QRLoginClient {
	jar, _ := cookiejar.New(nil)
	return &QRLoginClient{
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
			Jar:     jar,
		},
		userAgent: requester.UserAgent,
	}
}

// GetQRCode 获取登录二维码
func (c *QRLoginClient) GetQRCode() (*QRCodeInfo, error) {
	req, err := http.NewRequest(http.MethodGet, QRCodeGetURL, nil)
	if err != nil {
		return nil, fmt.Errorf("创建请求失败: %v", err)
	}
	req.Header.Set("User-Agent", c.userAgent)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("请求二维码失败: %v", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("读取响应失败: %v", err)
	}

	var qrResp QRCodeResponse
	if err := json.Unmarshal(body, &qrResp); err != nil {
		return nil, fmt.Errorf("解析响应失败: %v", err)
	}

	if qrResp.Errno != 0 {
		return nil, fmt.Errorf("获取二维码失败: errno=%d", qrResp.Errno)
	}

	// 构建完整的二维码图片URL
	imgURL := qrResp.ImgURL
	if !strings.HasPrefix(imgURL, "http") {
		imgURL = "https://" + imgURL
	}

	// 解码图片获取真实二维码内容，用于生成可扫描的ASCII二维码
	var asciiQR string
	qrContent, err := decodeQRCodeFromURL(imgURL)
	if err == nil && qrContent != "" {
		// 使用真实内容生成ASCII二维码
		asciiQR = generateASCIIQRCode(qrContent)
	} else {
		// 解码失败时，使用URL生成（备用方案，扫描后会跳转）
		asciiQR = generateASCIIQRCode(imgURL)
	}

	return &QRCodeInfo{
		Sign:     qrResp.Sign,
		ImgURL:   imgURL,
		ASCIIQA:  asciiQR,
		ExpireAt: time.Now().Add(DefaultQRCodeTimeout),
	}, nil
}

// CheckQRStatus 检查二维码扫描状态
func (c *QRLoginClient) CheckQRStatus(sign string) (*QRLoginResult, error) {
	url := fmt.Sprintf(QRCodeChannelURL, sign)

	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("创建请求失败: %v", err)
	}
	req.Header.Set("User-Agent", c.userAgent)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("请求状态失败: %v", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("读取响应失败: %v", err)
	}

	respStr := string(body)
	// 去除JSONP回调包装
	respStr = strings.TrimPrefix(respStr, "(")
	respStr = strings.TrimSuffix(respStr, ")")

	var channelResp ChannelResponse
	if err := json.Unmarshal([]byte(respStr), &channelResp); err != nil {
		// 可能是正在等待，返回等待状态
		return &QRLoginResult{
			Status:  QRStatusWaiting,
			Message: "等待扫码",
		}, nil
	}

	if channelResp.Errno != 0 {
		return &QRLoginResult{
			Status:  QRStatusWaiting,
			Message: "等待扫码",
		}, nil
	}

	if channelResp.ChannelV == "" {
		return &QRLoginResult{
			Status:  QRStatusWaiting,
			Message: "等待扫码",
		}, nil
	}

	// 解析channel_v中的JSON
	var channelData ChannelData
	// 处理可能的转义字符
	channelV := strings.ReplaceAll(channelResp.ChannelV, "\\", "")
	if err := json.Unmarshal([]byte(channelV), &channelData); err != nil {
		// 尝试直接解析
		if err := json.Unmarshal([]byte(channelResp.ChannelV), &channelData); err != nil {
			return &QRLoginResult{
				Status:  QRStatusError,
				Message: fmt.Sprintf("解析状态数据失败: %v", err),
			}, nil
		}
	}

	// status=0 表示已确认登录
	if channelData.Status == 0 && channelData.V != "" {
		return &QRLoginResult{
			Status:    QRStatusConfirmed,
			TempBDUSS: channelData.V,
			Message:   "扫码确认成功",
		}, nil
	}

	// 其他状态可能是已扫码等待确认
	return &QRLoginResult{
		Status:  QRStatusScanned,
		Message: "已扫码，请在手机上确认登录",
	}, nil
}

// ExchangeBDUSS 使用临时BDUSS交换正式凭证
func (c *QRLoginClient) ExchangeBDUSS(tempBDUSS string) (bduss, stoken string, err error) {
	url := fmt.Sprintf(QRCodeLoginURL, tempBDUSS)

	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return "", "", fmt.Errorf("创建请求失败: %v", err)
	}
	req.Header.Set("User-Agent", c.userAgent)

	// 不要自动跟随重定向，以便获取Set-Cookie
	c.httpClient.CheckRedirect = func(req *http.Request, via []*http.Request) error {
		return http.ErrUseLastResponse
	}
	defer func() {
		c.httpClient.CheckRedirect = nil
	}()

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return "", "", fmt.Errorf("请求登录失败: %v", err)
	}
	defer resp.Body.Close()

	// 从Set-Cookie中提取BDUSS和STOKEN
	cookies := resp.Header.Values("Set-Cookie")
	for _, cookie := range cookies {
		if strings.Contains(cookie, "BDUSS=") {
			re := regexp.MustCompile(`BDUSS=([^;]+)`)
			matches := re.FindStringSubmatch(cookie)
			if len(matches) >= 2 {
				bduss = matches[1]
			}
		}
		if strings.Contains(cookie, "STOKEN=") {
			re := regexp.MustCompile(`STOKEN=([^;]+)`)
			matches := re.FindStringSubmatch(cookie)
			if len(matches) >= 2 {
				stoken = matches[1]
			}
		}
	}

	if bduss == "" {
		return "", "", fmt.Errorf("未能从响应中获取BDUSS")
	}

	return bduss, stoken, nil
}

// generateASCIIQRCode 生成ASCII格式二维码
func generateASCIIQRCode(content string) string {
	var buf bytes.Buffer
	config := qrterminal.Config{
		Level:     qrterminal.L,
		Writer:    &buf,
		BlackChar: qrterminal.BLACK,
		WhiteChar: qrterminal.WHITE,
		QuietZone: 1,
	}
	qrterminal.GenerateWithConfig(content, config)
	return buf.String()
}

// decodeQRCodeFromURL 从URL下载二维码图片并解码获取内容
func decodeQRCodeFromURL(imgURL string) (string, error) {
	// 下载图片
	resp, err := http.Get(imgURL)
	if err != nil {
		return "", fmt.Errorf("下载二维码图片失败: %v", err)
	}
	defer resp.Body.Close()

	// 解码图片
	img, _, err := image.Decode(resp.Body)
	if err != nil {
		return "", fmt.Errorf("解码图片失败: %v", err)
	}

	// 使用zxing解析二维码
	bmp, err := gozxing.NewBinaryBitmapFromImage(img)
	if err != nil {
		return "", fmt.Errorf("转换图片失败: %v", err)
	}

	reader := qrcode.NewQRCodeReader()
	result, err := reader.Decode(bmp, nil)
	if err != nil {
		return "", fmt.Errorf("解析二维码失败: %v", err)
	}

	return result.GetText(), nil
}

// printQRCodeToTerminal 在终端打印二维码
func printQRCodeToTerminal(imgURL string) {
	fmt.Println()
	fmt.Println("请使用百度网盘APP扫描以下二维码登录：")
	fmt.Println()

	// 尝试解码远程二维码图片获取真实内容
	qrContent, err := decodeQRCodeFromURL(imgURL)
	if err != nil {
		// 如果解码失败，提示用户打开链接
		fmt.Printf("无法生成终端二维码: %v\n", err)
		fmt.Printf("请打开以下链接查看并扫描二维码：\n%s\n\n", imgURL)
		return
	}

	// 使用真实的二维码内容生成终端二维码
	qrterminal.GenerateHalfBlock(qrContent, qrterminal.L, os.Stdout)
	fmt.Println()
	fmt.Printf("或打开以下链接查看二维码：\n%s\n\n", imgURL)
}

// RunQRLogin 执行扫码登录（CLI入口）
func RunQRLogin(timeout time.Duration) error {
	if timeout <= 0 {
		timeout = DefaultQRCodeTimeout
	}

	client := NewQRLoginClient()

	// 1. 获取二维码
	fmt.Println("正在获取登录二维码...")
	qrInfo, err := client.GetQRCode()
	if err != nil {
		return fmt.Errorf("获取二维码失败: %v", err)
	}

	// 2. 在终端显示二维码
	printQRCodeToTerminal(qrInfo.ImgURL)

	// 3. 轮询扫码状态
	fmt.Println("等待扫码...")
	startTime := time.Now()
	ticker := time.NewTicker(PollInterval)
	defer ticker.Stop()

	var lastStatus QRCodeStatus = QRStatusWaiting
	for {
		select {
		case <-ticker.C:
			if time.Since(startTime) > timeout {
				return fmt.Errorf("二维码已过期，请重新运行 qrlogin 命令")
			}

			result, err := client.CheckQRStatus(qrInfo.Sign)
			if err != nil {
				continue // 忽略错误，继续轮询
			}

			// 状态变更时打印提示
			if result.Status != lastStatus {
				lastStatus = result.Status
				switch result.Status {
				case QRStatusScanned:
					fmt.Println("已扫码，请在手机上确认登录...")
				case QRStatusConfirmed:
					fmt.Println("扫码确认成功，正在完成登录...")
				}
			}

			// 已确认登录
			if result.Status == QRStatusConfirmed && result.TempBDUSS != "" {
				// 4. 交换正式凭证
				bduss, stoken, err := client.ExchangeBDUSS(result.TempBDUSS)
				if err != nil {
					return fmt.Errorf("交换凭证失败: %v", err)
				}

				// 5. 保存用户配置
				baidu, err := pcsconfig.Config.SetupUserByBDUSS(bduss, "", stoken, "")
				if err != nil {
					return fmt.Errorf("保存用户配置失败: %v", err)
				}

				fmt.Printf("\n登录成功！用户名: %s\n", baidu.Name)
				return nil
			}
		}
	}
}
