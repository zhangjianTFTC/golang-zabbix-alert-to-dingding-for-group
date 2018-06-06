package main

import (
	"encoding/json"
	"fmt"
	"bytes"
	"net/http"
	"os"
	"io/ioutil"
	"runtime"
	"time"
	"encoding/xml"
	"flag"
)

/*
zabbix报警信息
 */
type Alert struct {
	From                   string `json:"from" xml:"from"`
	Time                   string `json:"time" xml:"time"`
	Level                  string `json:"level" xml:"level"`
	Name                   string `json:"name" xml:"name"`
	Key                    string `json:"key" xml:"key"`
	Value                  string `json:"value" xml:"value"`
	Now                    string `json:"now" xml:"now"`
	ID                     string `json:"id" xml:"id"`
	IP                     string `json:"ip" xml:"ip"`
	Color                  string `json:"color" xml:"color"`
	Url                    string `json:"url" xml:"url"`
	Age                    string `json:"age" xml:"age"`
	Status                 string `json:"status" xml:"status"`
	RecoveryTime           string `json:"recoveryTime" xml:"recoveryTime"`
	Acknowledgement        string `json:"acknowledgement" xml:"acknowledgement"`
	Acknowledgementhistory string `json:"acknowledgementhistory" xml:"acknowledgementhistory"`
}

/*
发送给钉钉的信息
 */
type DingMsg struct {
	Msgtype     string `json:"msgtype"`
	Markdown    struct {
		Title      string `json:"title"`
		Text       string `json:"text"`
	} `json:"markdown"`
}

/*
zabbix需要传送的参数
 */
type MsgInfo struct {
	//消息属性和内容
	Webhook, Msg, Url, log, Style string
}

var msgInfo MsgInfo
var logPath string

func log(message interface{}) {
	pc, file, line, _ := runtime.Caller(1)

	f := runtime.FuncForPC(pc).Name()
	now := time.Now().Format("15:04:05.000")
	date := time.Now().Format("2006-01-02")

	str := fmt.Sprintf("%s %s:%d [%s]: %v", now, file, line, f, message)

	fmt.Println(str)
	if logPath != "" {
		fname := fmt.Sprintf("%s/zabbix_to_dingding_%s.log", logPath, date)
		logfile, err := os.OpenFile(fname, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
		defer logfile.Close()
		if err != nil {
			fmt.Printf("%s %s:%d [%s]: [日志创建错误] %v\r\n", now, file, line, f, err)
		}
		logfile.WriteString(str + "\r\n")
	}
}

/*
构造发送给dingding的json
 */
func makeMsg(msg string) string {
	//	根据json或xml文本创建消息体
	log("开始创建消息。")
	log(fmt.Sprintf(`来源消息：
%v`, msg))

	var alert Alert
	if msgInfo.Style == "xml" {
		log("来源消息格式为XML。")
		err := xml.Unmarshal([]byte(msg), &alert)
		if err != nil {
			log(fmt.Sprintf("XML 解析失败：%v", err))
			os.Exit(1)
		}
	} else if msgInfo.Style == "json" {
		log("来源消息格式为Json。")
		err := json.Unmarshal([]byte(msg), &alert)
		if err != nil {
			log(fmt.Sprintf("Json 解析失败：%v", err))
			os.Exit(1)
		}
	} else {
		log("未指定来源消息格式，默认使用Json解析。")
		err := json.Unmarshal([]byte(msg), &alert)
		if err != nil {
			log(fmt.Sprintf("Json 解析失败：%v", err))
			os.Exit(1)
		}
	}
	log("来源消息解析成功。")
	var dingMsg DingMsg
	//给dingMsg各元素赋值
	dingMsg.Msgtype = "markdown"
	if alert.Status == "PROBLEM" {
		dingMsg.Markdown.Title = "故障：" + alert.Name
	} else if alert.Status == "OK" {
		dingMsg.Markdown.Title = "恢复：" + alert.Name
	} else if alert.Status == "RESOLVED" {
		dingMsg.Markdown.Title = "恢复：" + alert.Name
	} else {
		dingMsg.Markdown.Title = alert.Name
	}
	dingMsg.Markdown.Text  = "### " + alert.Name +
		" \n >" +
		//" \n ## 出现故障" +
		" \n ##### 故障时间：" + alert.Time +
		" \n ##### 恢复时间：" + alert.RecoveryTime +
		" \n ##### 故障时长：" + alert.Age +
		" \n ##### 主机名：" + alert.From +
		" \n ##### IP地址：" + alert.IP +
		" \n ##### 检测项：" + alert.Key +
		" \n ## **" + alert.Value + "**" +
		" \n ##### " + fmt.Sprintf("[%s·%s(%s)]", alert.From, "恢复", alert.ID) +
		" \n ##### 详情：[请点击](" + msgInfo.Url + ")"

	//if alert.Url != "" {
	//	dingMsg.Oa.MessageURL = alert.Url
	//}

	/*dingMsg.Oa.Head.Bgcolor = alert.Color
	dingMsg.Oa.Body.Title = alert.Name
	dingMsg.Oa.Body.Form[0].Key = "告警级别："
	dingMsg.Oa.Body.Form[1].Key = "故障时间："
	dingMsg.Oa.Body.Form[2].Key = "故障时长："
	dingMsg.Oa.Body.Form[3].Key = "IP地址："
	dingMsg.Oa.Body.Form[4].Key = "检测项："
	dingMsg.Oa.Body.Form[0].Value = alert.Level
	dingMsg.Oa.Body.Form[1].Value = alert.Time
	dingMsg.Oa.Body.Form[2].Value = alert.Age
	dingMsg.Oa.Body.Form[3].Value = alert.IP
	dingMsg.Oa.Body.Form[4].Value = alert.Key
	dingMsg.Oa.Body.Rich.Num = alert.Now
	if alert.Status == "PROBLEM" {
		//  故障处理
		dingMsg.Oa.Body.Author = fmt.Sprintf("[%s·%s(%s)]", alert.From, "故障", alert.ID)
		if strings.Replace(alert.Acknowledgement, " ", "", -1) == "Yes" {
			dingMsg.Oa.Body.Content = "故障已经被确认，" + alert.Acknowledgementhistory
		}
	} else if alert.Status == "OK" {
		//  恢复处理
		dingMsg.Oa.Body.Form[0].Key = "故障时间："
		dingMsg.Oa.Body.Form[1].Key = "恢复时间："
		dingMsg.Oa.Body.Form[0].Value = alert.Time
		dingMsg.Oa.Body.Form[1].Value = alert.RecoveryTime
		dingMsg.Oa.Body.Author = fmt.Sprintf("[%s·%s(%s)]", alert.From, "恢复", alert.ID)

	} else if alert.Status == "RESOLVED" {
		//  恢复处理
		dingMsg.Oa.Body.Form[0].Key = "故障时间："
		dingMsg.Oa.Body.Form[1].Key = "恢复时间："
		dingMsg.Oa.Body.Form[0].Value = alert.Time
		dingMsg.Oa.Body.Form[1].Value = alert.RecoveryTime
		dingMsg.Oa.Body.Author = fmt.Sprintf("[%s·%s(%s)]", alert.From, "恢复", alert.ID)

	} else if alert.Status == "msg" {
		dingMsg.Oa.Body.Title = alert.Name
		dingMsg.Oa.Body.Form[0].Key = ""
		dingMsg.Oa.Body.Form[1].Key = ""
		dingMsg.Oa.Body.Form[2].Key = ""
		dingMsg.Oa.Body.Form[3].Key = ""
		dingMsg.Oa.Body.Form[4].Key = ""
		dingMsg.Oa.Body.Form[0].Value = ""
		dingMsg.Oa.Body.Form[1].Value = ""
		dingMsg.Oa.Body.Form[2].Value = ""
		dingMsg.Oa.Body.Form[3].Value = ""
		dingMsg.Oa.Body.Form[4].Value = ""
		dingMsg.Oa.Body.Content = alert.Acknowledgementhistory
	} else {
		//  其他status状况处理
		dingMsg.Oa.MessageURL = "https://www.qiansw.com/golang-zabbix-alter-to-dingding.html"
		dingMsg.Oa.Body.Content = "ZABBIX动作配置有误，请至甜菜网[qiansw.com]或直接[点击此消息]查看具体配置文档。"
		dingMsg.Oa.Body.Author = fmt.Sprintf("[%s·%s(%s)]", alert.From, alert.Status, alert.ID)
		if strings.Replace(alert.Acknowledgement, " ", "", -1) == "Yes" {
			dingMsg.Oa.Body.Content = "故障已经被确认，" + alert.Acknowledgementhistory
		}
	}*/
	//	创建post给钉钉的Json文本
	JsonMsg, err := json.Marshal(dingMsg)
	if err != nil {
		log(err)
		os.Exit(1)
	}
	log(fmt.Sprintf("消息创建完成：%s\r\n", string(JsonMsg)))
	return string(JsonMsg)
}

func sendMsg(msg string) (status bool) { //发送OA消息，,返回成功或失败
	//log(fmt.Sprintf("需要POST的内容：%v", msg))
	body := bytes.NewBuffer([]byte(msg))
	url := msgInfo.Webhook
	//	fmt.Println(url)
	res, err := http.Post(url, "application/json;charset=utf-8", body)
	if err != nil {
		log(err)
		os.Exit(1)
		return
	}
	result, err := ioutil.ReadAll(res.Body)
	res.Body.Close()
	if err != nil {
		log(err)
		os.Exit(1)
		return
	}
	//log(fmt.Sprintf("钉钉接口返回消息：%s", result))
	fmt.Sprintf("钉钉接口返回消息：%s", result)
	return
}

func init() {

	flag.StringVar(&msgInfo.Webhook, "webhook", "", "消息的接收人，可以在钉钉后台查看，可空。")
	flag.StringVar(&msgInfo.Msg, "msg", `{ "from": "甜菜网", "time": "2016.07.28 17:00:05", "level": "Warning", "name": "这是一个甜菜网（qiansw.com）提供的ZABBIX钉钉报警插件。", "key": "icmpping", "value": "30ms", "now": "56ms", "id": "1637", "ip": "8.8.8.8", "color":"FF4A934A", "age":"3m", "recoveryTime":"2016.07.28 17:03:05", "status":"OK" }`, "Json格式的文本消息内容，不可空。")
	flag.StringVar(&msgInfo.Url, "url", "http://www.itiancai.com", "消息内容点击后跳转到的URL，可空。")
	flag.StringVar(&msgInfo.Style, "style", "json", "Msg的格式，可选json和xml，推荐使用xml（支持消息中含双引号），可空。")
	flag.StringVar(&logPath, "log", "", "指定存放 log 的目录，不指定则不记录 log。")
	flag.Parse()

	pc, file, line, _ := runtime.Caller(1)

	f := runtime.FuncForPC(pc).Name()
	now := time.Now().Format("15:04:05.000")
	date := time.Now().Format("2006-01-02")

	if logPath != "" {
		fname := fmt.Sprintf("%s/zabbix_to_dingding_%s.log", logPath, date)
		logfile, err := os.OpenFile(fname, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
		defer logfile.Close()
		if err != nil {
			fmt.Printf("%s %s:%d [%s]: [日志创建错误] %v\r\n", now, file, line, f, err)
			os.Exit(1)
		}
		logfile.WriteString("程序启动……" + "\r\n")
	}
	log("初始化完成。")
}

func main() {

	p := &DingMsg{}
	p.Msgtype           = "markdown"
	//p.Text.Content    = "这是一个测试消息!"

	//p.At.AtMobiles[0] = "123"
	//p.At.AtMobiles[1] = "456"
	//p.At.IsAtAll      = false
	p.Markdown.Title    = "故障：Free disk space is less than 20% on volume /data"
	p.Markdown.Text     = "### Free disk space is less than 20% on volume /data" +
		" \n >" +
		" \n ## 出现故障" +
		" \n ##### 故障时间：2018.05.30 19:26:39" +
		" \n ##### 恢复时间：2018.05.30 20:17:39" +
		" \n ##### 故障时长：51m" +
		" \n ##### 主机名：xxx.xxx.xxx.xxx" +
		" \n ##### IP地址：10.10.0.1" +
		" \n ##### 检测项：vfs.fs.size[/data,pfree]" +
		" \n ## **22.99 %**" +
		" \n ##### [xxx.xxx.xxx.xxx·恢复(6785)]5月30日 20:18" +
		" \n ##### 详情：[请点击](http://www.thinkpage.cn/)"

	data, _ := json.Marshal(p)
	fmt.Print(string(data))

	//sendMsg(string(data))
	sendMsg(makeMsg(msgInfo.Msg))
}