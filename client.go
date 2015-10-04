// client.go
package main
// import libraries
import (
    "fmt"
    "log"
    "net"
    "net/rpc/jsonrpc"
    "strconv"
    "strings"
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

type CheckResponse struct {
    Stocks string
    CMV float32
    UnvestedAmount float32
}

type CheckRequest struct {
    TId int
}
// main function
func main() {
    client, err := net.Dial("tcp", "127.0.0.1:1234")
    if err != nil {
        log.Fatal("dialing:", err)
    }
    var stockSymbolAndPercentage string
    var budget float32
    // input from user
    // stockSymbolAndPercentage and budget
    fmt.Scanf("%s", &stockSymbolAndPercentage)
    fmt.Scanf("%f", &budget)
    args := &Args{stockSymbolAndPercentage,budget}
    var reply Reply
    var resp CheckResponse
    c := jsonrpc.NewClient(client)
    if args.B == 0 {
        // convert string to integer
        t,_ :=strconv.Atoi(stockSymbolAndPercentage)
        req := &CheckRequest{t}
        // call function on server
        err = c.Call("Stock.Check", req, &resp)
        // display the output on console
        fmt.Printf("\"stocks\":%s\n\"currentMarketValue\":%f\n\"unvestedAmount\":%f", resp.Stocks, resp.CMV, resp.UnvestedAmount)
    } else {
        // tokenize the stockSymbolAndPercentage variable to verify the percentage used
        token := strings.FieldsFunc(args.Sp,func(r rune) bool {
            return r ==':' || r==','
            })
        var total float32
        for i := 0; i < len(token); i++ {
            i++
            token[i]=strings.TrimSuffix(token[i], "%")
            // convert the string to float32
            percentage64, _:= strconv.ParseFloat(token[i], 32)
            total = total + float32(percentage64)
        }
        // condition to check total percentage of budget used is 100% or not
        if total == 100 {
            err = c.Call("Stock.Buy", args, &reply)        
            fmt.Printf("\"tradeId\":%d\n\"stocks\":%s\n\"unvestedAmount\":%f", reply.TradeId, reply.Stocks, reply.UnvestedAmount)
        } else {
            log.Fatal("Error:Not 100% budget used")    
        }
    }    
    if err != nil {
        log.Fatal("arith error:", err)
    }
}