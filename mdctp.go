package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/pseudocodes/go2ctp/ctp"
	"github.com/pseudocodes/go2ctp/thost"
)

type MdCtp struct {
	ctp.BaseMdSpi
	UserID   string
	BrokerID string
	mdapi    thost.MdApi
	resultC  chan int
}

var _ thost.MdSpi = &MdCtp{}

func CreateMdCtp(userID, brokerID string) *MdCtp {
	mdapi := ctp.CreateMdApi(ctp.MdFlowPath("flows/"), ctp.MdUsingUDP(false), ctp.MdMultiCast(false))

	mdctp := &MdCtp{
		UserID:   userID,
		BrokerID: brokerID,
		mdapi:    mdapi,
		resultC:  make(chan int, 1),
	}
	return mdctp
}

func (mdctp *MdCtp) Connect(frontAddr string) error {
	mdctp.mdapi.RegisterSpi(mdctp)
	mdctp.mdapi.RegisterFront(frontAddr)
	mdctp.mdapi.Init()
	ret := <-mdctp.resultC
	if ret != 0 {
		log.Printf("Connect failed: %d", ret)
		return fmt.Errorf("Connect failed: %d", ret)
	} else {
		log.Printf("Connect success")
	}
	return nil // 返回错误
}

// Login 用户登录
func (mdctp *MdCtp) Login() error {
	loginReq := &thost.CThostFtdcReqUserLoginField{}
	copy(loginReq.UserID[:], mdctp.UserID)
	copy(loginReq.Password[:], "")
	copy(loginReq.BrokerID[:], mdctp.BrokerID)

	ret := mdctp.mdapi.ReqUserLogin(loginReq, 1)
	if ret != 0 {
		return fmt.Errorf("登录请求发送失败，返回码: %d", ret)
	}

	log.Printf("发送登录请求: UserID=%s, BrokerID=%s\n", mdctp.UserID, mdctp.BrokerID)
	ret = <-mdctp.resultC
	if ret != 0 {
		return fmt.Errorf("登录失败，返回码: %d", ret)
	}
	return nil
}

// Logout 用户登出
func (mdctp *MdCtp) Logout(userID, brokerID string) error {
	logoutReq := &thost.CThostFtdcUserLogoutField{}
	copy(logoutReq.UserID[:], userID)
	copy(logoutReq.BrokerID[:], brokerID)

	ret := mdctp.mdapi.ReqUserLogout(logoutReq, 2)
	if ret != 0 {
		return fmt.Errorf("登出请求发送失败，返回码: %d", ret)
	}

	log.Printf("发送登出请求: UserID=%s, BrokerID=%s\n", userID, brokerID)
	ret = <-mdctp.resultC
	if ret != 0 {
		return fmt.Errorf("登出失败，返回码: %d", ret)
	}
	return nil
}

// SubscribeMarketData 订阅行情数据
func (mdctp *MdCtp) SubscribeMarketData(instrumentIDs ...string) error {
	if len(instrumentIDs) == 0 {
		return fmt.Errorf("合约列表为空")
	}

	ret := mdctp.mdapi.SubscribeMarketData(instrumentIDs...)
	if ret != 0 {
		log.Printf("订阅行情失败: %+v, 返回码: %d\n", instrumentIDs, ret)
	} else {
		log.Printf("订阅行情成功: %+v\n", instrumentIDs)
	}

	log.Printf("批量订阅行情: %+v\n", instrumentIDs)
	ret = <-mdctp.resultC
	if ret != 0 {
		return fmt.Errorf("订阅行情失败，返回码: %d", ret)
	}
	return nil
}

// UnsubscribeMarketData 取消订阅行情数据
func (mdctp *MdCtp) UnsubscribeMarketData(instrumentIDs ...string) error {
	if len(instrumentIDs) == 0 {
		return fmt.Errorf("合约列表为空")
	}

	ret := mdctp.mdapi.UnSubscribeMarketData(instrumentIDs...)
	if ret != 0 {
		log.Printf("取消订阅行情失败: %+v, 返回码: %d", instrumentIDs, ret)
	} else {
		log.Printf("取消订阅行情成功: %+v", instrumentIDs)
	}

	log.Printf("批量取消订阅行情: %+v", instrumentIDs)
	ret = <-mdctp.resultC
	if ret != 0 {
		return fmt.Errorf("取消订阅行情失败，返回码: %d", ret)
	}
	return nil
}

// Release 释放资源
func (mdctp *MdCtp) Release() {
	if mdctp.mdapi != nil {
		mdctp.mdapi.Release()
		log.Println("MdCtp 资源已释放")
	}
}

func (mdctp *MdCtp) OnFrontConnected() {
	log.Println("OnFrontConnected")
	mdctp.resultC <- 0
}

func (mdctp *MdCtp) OnFrontDisconnected(reason int) {
	log.Println("OnFrontDisconnected", reason)
}

// OnHeartBeatWarning 当客户端与交易后台通信连接断开时，该方法被调用。
func (mdctp *MdCtp) OnHeartBeatWarning(timelapse int) {
	log.Printf("OnHeartBeatWarning: 心跳超时 %d 秒", timelapse)
}

func (mdctp *MdCtp) OnRspUserLogin(userLogin *thost.CThostFtdcRspUserLoginField, rspInfo *thost.CThostFtdcRspInfoField, nRequestID int, bIsLast bool) {
	if rspInfo != nil && rspInfo.ErrorID != 0 {
		log.Printf("OnRspUserLogin 失败: ErrorID=%d, ErrorMsg=%s", rspInfo.ErrorID, rspInfo.ErrorMsg)
		mdctp.resultC <- int(rspInfo.ErrorID)
	} else {
		log.Printf("OnRspUserLogin 成功: UserID=%s, BrokerID=%s", userLogin.UserID.String(), userLogin.BrokerID.String())
		mdctp.resultC <- 0
	}
}

// OnRspUserLogout 登出请求响应
func (mdctp *MdCtp) OnRspUserLogout(userLogout *thost.CThostFtdcUserLogoutField, rspInfo *thost.CThostFtdcRspInfoField, nRequestID int, bIsLast bool) {
	if rspInfo != nil && rspInfo.ErrorID != 0 {
		log.Printf("OnRspUserLogout 失败: ErrorID=%d, ErrorMsg=%s", rspInfo.ErrorID, rspInfo.ErrorMsg)
		mdctp.resultC <- int(rspInfo.ErrorID)
	} else {
		log.Printf("OnRspUserLogout 成功: UserID=%s", userLogout.UserID)
		mdctp.resultC <- 0
	}
}

// OnRspError 错误应答
func (mdctp *MdCtp) OnRspError(rspInfo *thost.CThostFtdcRspInfoField, nRequestID int, bIsLast bool) {
	if rspInfo != nil {
		log.Printf("OnRspError: ErrorID=%d, ErrorMsg=%s, RequestID=%d, IsLast=%v",
			rspInfo.ErrorID, rspInfo.ErrorMsg, nRequestID, bIsLast)
	}
}

// OnRspSubMarketData 订阅行情应答
func (mdctp *MdCtp) OnRspSubMarketData(specificInstrument *thost.CThostFtdcSpecificInstrumentField, rspInfo *thost.CThostFtdcRspInfoField, nRequestID int, bIsLast bool) {
	if rspInfo != nil && rspInfo.ErrorID != 0 {
		log.Printf("订阅行情失败: InstrumentID=%s, ErrorID=%d, ErrorMsg=%s",
			specificInstrument.InstrumentID, rspInfo.ErrorID, rspInfo.ErrorMsg)
		mdctp.resultC <- int(rspInfo.ErrorID)
	} else {
		log.Printf("订阅行情成功: InstrumentID=%s", specificInstrument.InstrumentID)
		mdctp.resultC <- 0
	}
}

// OnRspUnSubMarketData 取消订阅行情应答
func (mdctp *MdCtp) OnRspUnSubMarketData(specificInstrument *thost.CThostFtdcSpecificInstrumentField, rspInfo *thost.CThostFtdcRspInfoField, nRequestID int, bIsLast bool) {
	if rspInfo != nil && rspInfo.ErrorID != 0 {
		log.Printf("取消订阅行情失败: InstrumentID=%s, ErrorID=%d, ErrorMsg=%s",
			specificInstrument.InstrumentID, rspInfo.ErrorID, rspInfo.ErrorMsg)
		mdctp.resultC <- int(rspInfo.ErrorID)
	} else {
		log.Printf("取消订阅行情成功: InstrumentID=%s", specificInstrument.InstrumentID)
		mdctp.resultC <- 0
	}
}

// Instrument 表示 API 返回的单个合约信息。
type Instrument struct {
	ExchangeID               string   `json:"ExchangeID"`
	InstrumentID             string   `json:"InstrumentID"`
	InstrumentName           string   `json:"InstrumentName"`
	ProductClass             string   `json:"ProductClass"`
	ProductID                string   `json:"ProductID"`
	VolumeMultiple           int      `json:"VolumeMultiple"`
	PriceTick                float64  `json:"PriceTick"`
	LongMarginRatioByMoney   float64  `json:"LongMarginRatioByMoney"`
	ShortMarginRatioByMoney  float64  `json:"ShortMarginRatioByMoney"`
	LongMarginRatioByVolume  float64  `json:"LongMarginRatioByVolume"`
	ShortMarginRatioByVolume float64  `json:"ShortMarginRatioByVolume"`
	OpenRatioByMoney         float64  `json:"OpenRatioByMoney"`
	OpenRatioByVolume        float64  `json:"OpenRatioByVolume"`
	CloseRatioByMoney        float64  `json:"CloseRatioByMoney"`
	CloseRatioByVolume       float64  `json:"CloseRatioByVolume"`
	CloseTodayRatioByMoney   float64  `json:"CloseTodayRatioByMoney"`
	CloseTodayRatioByVolume  float64  `json:"CloseTodayRatioByVolume"`
	DeliveryYear             int      `json:"DeliveryYear"`
	DeliveryMonth            int      `json:"DeliveryMonth"`
	OpenDate                 string   `json:"OpenDate"`
	ExpireDate               string   `json:"ExpireDate"`
	DeliveryDate             string   `json:"DeliveryDate"`
	UnderlyingInstrID        string   `json:"UnderlyingInstrID"`
	UnderlyingMultiple       int      `json:"UnderlyingMultiple"`
	OptionsType              string   `json:"OptionsType"`
	StrikePrice              *float64 `json:"StrikePrice"` // 指针类型处理 null 值
	InstLifePhase            string   `json:"InstLifePhase"`
}

// InstrumentsResponse 表示 API 的完整 JSON 响应结构。
type InstrumentsResponse struct {
	RspCode    int          `json:"rsp_code"`
	RspMessage string       `json:"rsp_message"`
	Data       []Instrument `json:"data"`
}

// GetInstruments 从 OpenCTP 字典 API 获取合约数据。
//
// 参数:
//
//	types: 可选，商品类型切片，如 {"futures"}, {"futures", "option"}。
//	       有效值：stock、bond、fund、futures、option。注意 'futures' 是复数。
//	areas: 可选，国家/地区切片，如 {"China"}, {"China", "USA"}。
//	markets: 可选，交易所 ID 切片，如 {"SHFE"}, {"SHFE", "CFFEX"}。
//	products: 可选，品种 ID 切片，如 {"au"}, {"au", "rb", "IF"}。
//
// 返回值:
//
//	一个 InstrumentsResponse 结构体指针，包含 API 响应数据；如果请求失败，则返回 error。
func GetInstruments(types, areas, markets, products []string) (*InstrumentsResponse, error) {
	baseURL := "http://dict.openctp.cn/instruments"
	u, err := url.Parse(baseURL)
	if err != nil {
		return nil, fmt.Errorf("解析基础 URL 失败: %w", err)
	}

	q := u.Query()

	// 将字符串切片转换为逗号分隔的字符串
	if len(types) > 0 {
		q.Set("types", strings.Join(types, ","))
	}
	if len(areas) > 0 {
		q.Set("areas", strings.Join(areas, ","))
	}
	if len(markets) > 0 {
		q.Set("markets", strings.Join(markets, ","))
	}
	if len(products) > 0 {
		q.Set("products", strings.Join(products, ","))
	}
	u.RawQuery = q.Encode()

	// 创建一个带超时设置的 HTTP 客户端
	client := &http.Client{
		Timeout: 10 * time.Second, // 10 秒超时
	}

	resp, err := client.Get(u.String())
	if err != nil {
		return nil, fmt.Errorf("发起 HTTP 请求失败: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("API 返回非 OK 状态: %s", resp.Status)
	}

	var instrumentsResp InstrumentsResponse
	if err := json.NewDecoder(resp.Body).Decode(&instrumentsResp); err != nil {
		return nil, fmt.Errorf("解码 JSON 响应失败: %w", err)
	}

	return &instrumentsResp, nil
}

// ExampleMdCtpUsage 展示如何使用 MdCtp
func ExampleMdCtpUsage() {
	// 创建 MdCtp 实例
	mdctp := CreateMdCtp("your_user_id", "9999") // Linux 下的动态库路径

	// 连接到行情前置
	frontAddr := "tcp://180.168.146.187:10131" // 示例地址
	mdctp.Connect(frontAddr)

	// 等待连接建立后登录
	// 注意：实际使用中需要在 OnFrontConnected 回调中进行登录
	// userID := "your_user_id"
	// brokerID := "9999"

	err := mdctp.Login()
	if err != nil {
		log.Printf("登录失败: %v", err)
		return
	}

	// 订阅行情
	instruments := []string{"rb2508", "TA509", "CU2412"}
	err = mdctp.SubscribeMarketData(instruments...)
	if err != nil {
		log.Printf("订阅行情失败: %v\n", err)
	}

	// 示例：打印订阅的合约
	log.Printf("已订阅的合约: %v\n", instruments)

	// 程序结束时清理资源
	defer mdctp.Release()
}
