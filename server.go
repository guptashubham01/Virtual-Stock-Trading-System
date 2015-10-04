// server.go
package main
// import libraries
import (
    "net/http"
    "io/ioutil"
    "bytes"
    "strings"
    "strconv"
    "math"
    "log"
    "net"
    "net/rpc"
    "net/rpc/jsonrpc"
)

type Args struct {
    Sp string
    B float32
}

type Reply struct {
    TradeId int
    Stocks string
    UnvestedAmount float32
}

type stockDetails struct{
    name string
    percentage float32
    amount float32
    shareCount int
    price float32
}

type tradeDetails struct{
    tradeId int
    unvestedAmount float32
    stocks [5]stockDetails
}

type CheckResponse struct {
    Stocks string
    CMV float32
    UnvestedAmount float32
}

type CheckRequest struct {
    TId int
}

type Portfolio struct {
    StockPrice [5]float32
}

type Stock struct{}
var tradeCounter int
var trades [10]tradeDetails

// function for buying stocks
func (t *Stock) Buy(args *Args, reply *Reply) error {
    // tokenize the string
    token := strings.FieldsFunc(args.Sp,func(r rune) bool {
        return r ==':' || r==','
        })
    var buffer bytes.Buffer
    j:=0;
    // maintain the counter for tradeID
    tradeCounter++
    trades[tradeCounter].tradeId = tradeCounter
    trades[tradeCounter].unvestedAmount = 0
    // store name and percentage of each stock into the array 
    for i := 0; i < len(token); i++ {
        buffer.WriteString(token[i] + "+")
        trades[tradeCounter].stocks[j].name = token[i] 
        i++
        token[i]=strings.TrimSuffix(token[i], "%")
        percentage64, _:= strconv.ParseFloat(token[i], 32)
        trades[tradeCounter].stocks[j].percentage = float32(percentage64 / 100.00)
        trades[tradeCounter].stocks[j].amount = args.B * trades[tradeCounter].stocks[j].percentage
        j++
    }
    stockName := buffer.String()
    stockName = strings.TrimRight(stockName, "+")
    // creating url to get the requested stock price
    mainURL := "http://finance.yahoo.com/d/quotes.csv?s="
    var stockBuffer bytes.Buffer
    stockBuffer.WriteString(mainURL)
    stockBuffer.WriteString(stockName)
    info := "&f=nl1"
    stockBuffer.WriteString(info)
    url :=  stockBuffer.String()
    req, err := http.Get(url)
    if err != nil {
        panic(err)
    }
    body, _ := ioutil.ReadAll(req.Body)
    data:=string(body)
    shareData := strings.FieldsFunc(data,func(r rune) bool {
        return r==',' || r=='"'
        })
    j=0;
    // parse through the data to calculate the share count and unvested amount
    for i := 1; i < len(shareData); i++ {
        tr := strings.TrimSpace(shareData[i])
        price64, _ := strconv.ParseFloat(tr, 32)
        trades[tradeCounter].stocks[j].price = float32(price64)
        trades[tradeCounter].stocks[j].shareCount = int(math.Floor(float64(trades[tradeCounter].stocks[j].amount / trades[tradeCounter].stocks[j].price)))
        trades[tradeCounter].unvestedAmount = trades[tradeCounter].unvestedAmount + (trades[tradeCounter].stocks[j].amount - (float32(trades[tradeCounter].stocks[j].shareCount) * trades[tradeCounter].stocks[j].price))
        i++ 
        j++
    }
    // build the stocks quote string
    var share bytes.Buffer
    for i:=0; i<len(trades[tradeCounter].stocks) && trades[tradeCounter].stocks[i].percentage != 0; i++ {
        share.WriteString("\"")
        share.WriteString(strings.ToUpper(trades[tradeCounter].stocks[i].name))
        share.WriteString(":")
        share.WriteString(strconv.Itoa(trades[tradeCounter].stocks[i].shareCount))
        share.WriteString(":$")
        share.WriteString(strconv.FormatFloat(float64(trades[tradeCounter].stocks[i].price), 'f', -1, 32))
        share.WriteString("\",")
    }
    stockString := strings.TrimSuffix(share.String(), ",")
    reply.Stocks = stockString
    reply.TradeId = trades[tradeCounter].tradeId
    reply.UnvestedAmount = trades[tradeCounter].unvestedAmount
    return nil
}

// function to check the portfolio
func (t *Stock) Check(req *CheckRequest, resp *CheckResponse) error {
    var port Portfolio
    var share bytes.Buffer
    var shareUrl bytes.Buffer
    var mainUrl bytes.Buffer
    for i:=0; i<len(trades[req.TId].stocks) && trades[req.TId].stocks[i].percentage != 0; i++ {
        shareUrl.WriteString(trades[req.TId].stocks[i].name)
        shareUrl.WriteString("+")
    }
    tempString := shareUrl.String()
    UrlString := strings.TrimSuffix(tempString, "+")
    // url to find the updated price of the stocks bought before
    mainURL := "http://finance.yahoo.com/d/quotes.csv?s="
    info := "&f=nl1"
    mainUrl.WriteString(mainURL)
    mainUrl.WriteString(UrlString)
    mainUrl.WriteString(info)
    URL := mainUrl.String()
    result, err := http.Get(URL)
    if err != nil {
        panic(err)
    }
    body, _ := ioutil.ReadAll(result.Body)
    data:=string(body)

    shareData := strings.FieldsFunc(data,func(r rune) bool {
        return r==',' || r=='"'
        })
    j:=0;
    // store the price of each stock
    for i := 1; i < len(shareData); i++ {
        tr := strings.TrimSpace(shareData[i])
        price64, _ := strconv.ParseFloat(tr, 32)
        port.StockPrice[j]  = float32(price64)
        j++
        i++
    }
    var cmv float32
    // create string to return to client
    for i:=0; i<len(trades[req.TId].stocks) && trades[req.TId].stocks[i].percentage != 0; i++ {
        share.WriteString("\"")
        share.WriteString(strings.ToUpper(trades[req.TId].stocks[i].name))
        share.WriteString(":")
        share.WriteString(strconv.Itoa(trades[req.TId].stocks[i].shareCount))
        share.WriteString(":")
        if port.StockPrice[i] > trades[req.TId].stocks[i].price {
            share.WriteString("+")
        } else if port.StockPrice[i] < trades[req.TId].stocks[i].price {
            share.WriteString("-")
        } else {
            share.WriteString("")
        }
        share.WriteString("$")
        share.WriteString(strconv.FormatFloat(float64(port.StockPrice[i]), 'f', -1, 32))
        share.WriteString("\",")
        cmv = cmv + (port.StockPrice[i] * float32(trades[req.TId].stocks[i].shareCount))
    }
    stockString := strings.TrimSuffix(share.String(), ",")
    resp.Stocks = stockString
    resp.CMV = cmv
    resp.UnvestedAmount = trades[req.TId].unvestedAmount
    return nil
}

func main() {
    stock := new(Stock)
    server := rpc.NewServer()
    server.Register(stock)
    server.HandleHTTP(rpc.DefaultRPCPath, rpc.DefaultDebugPath)
    listener, e := net.Listen("tcp", ":1234")
    if e != nil {
        log.Fatal("listen error:", e)
    }
    for {
        if conn, err := listener.Accept(); err != nil {
            log.Fatal("accept error: " + err.Error())
        } else {
            log.Printf("new connection established\n")
            go server.ServeCodec(jsonrpc.NewServerCodec(conn))
        }
    }
}