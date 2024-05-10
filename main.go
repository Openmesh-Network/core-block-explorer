package main

import (
	"encoding/base64"
	"fmt"
	"html/template"
	"io"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	json "github.com/buger/jsonparser"
	"github.com/gokyle/filecache"
	"github.com/openmesh-network/core/bft/types"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protoreflect"
)

// Every new block update UI.
// Have a set of templates that get refreshed automatically.

// Show header crap like that.
// Show transactions!
// Show validators.
//	- Folding view?
// Add graphs?

type TemplateData struct {
	Hash         string
	PrevHash     string
	NextHash     string
	Height       int
	Validators   int
	BlockTime    []string
	Transactions template.HTML
	IsSummary    bool
	IsPrev       bool
}

const rpcUrl = "127.0.0.1:26655"
const renderDir = "renders"

var prevTemplate TemplateData

func newBlock(data []byte) {
	// When a new block is received recompile the data to their folders.

	fmt.Println("Got new block!")
	templateData := TemplateData{}
	{
		var err error
		templateData.Hash, err = json.GetString(data, "result", "block_id", "hash")
		templateData.PrevHash, err = json.GetString(data, "result", "block", "header", "last_block_id", "hash")
		hStr, err := json.GetString(data, "result", "block", "header", "height")
		templateData.Height, err = strconv.Atoi(hStr)

		if err != nil {
			panic(err)
		}

		// Parse the transactions as HTML:
		txsHtmlString := ""
		_, err = json.ArrayEach(data, func(value []byte, dataType json.ValueType, offset int, err error) {
			// Decode from base64

			rawTransaction, err := base64.StdEncoding.DecodeString(string(value))
			if err != nil {
				panic(err)
			}

			transaction := new(types.Transaction)
			proto.Unmarshal(rawTransaction, transaction)

			txsHtmlString += "<details><summary> " + types.TransactionType_name[int32(transaction.Type)] + " Transaction</summary>"

			txHtmlString := ""
			var reflectMsg protoreflect.Message
			if d := transaction.GetResourceData(); d != nil {
				reflectMsg = d.ProtoReflect()
			} else if d := transaction.GetVerificationData(); d != nil {
				reflectMsg = d.ProtoReflect()
			} else if d := transaction.GetNormalData(); d != nil {
				reflectMsg = d.ProtoReflect()
			} else if d := transaction.GetNodeRegistrationData(); d != nil {
				reflectMsg = d.ProtoReflect()
			} else {
				panic("Transaction type not handled!")
			}

			reflectMsg.Range(func(desc protoreflect.FieldDescriptor, val protoreflect.Value) bool {
				txHtmlString += "<table><tr>"
				txHtmlString += "<td>" + string(desc.Name()) + "</td>"
				txHtmlString += "<td>" + string(val.String()) + "</td>"
				txHtmlString += "</tr></table>"

				return true
			})

			txsHtmlString += txHtmlString
			txsHtmlString += "</details>"

		}, "result", "block", "data", "txs")

		templateData.Transactions = template.HTML(txsHtmlString)

		if err != nil {
			panic(err)
		}
	}

	{
		temp, err := template.ParseFiles("template.html")
		if err != nil {
			panic(err)
		}

		summaryFile, err := os.Create("renders/summary.html")
		if err != nil {
			panic(err)
		}
		blockFile, err := os.Create("renders/block/id/" + templateData.Hash + ".html")
		if err != nil {
			panic(err)
		}
		blockPrevFile, err := os.Create("renders/block/id/" + templateData.PrevHash + ".html")
		if err != nil {
			panic(err)
		}

		templateData.IsSummary = true
		err = temp.Execute(summaryFile, templateData)
		if err != nil {
			panic(err)
		}

		templateData.IsSummary = false
		temp.Execute(blockFile, templateData)
		if err != nil {
			panic(err)
		}
		prevTemplate.IsPrev = true
		prevTemplate.NextHash = templateData.Hash
		err = temp.Execute(blockPrevFile, prevTemplate)
		if err != nil {
			panic(err)
		}
	}

	{ // Parse transactions.
		// Take transaction from base64 to transaction and unmarshal.
	}

	prevTemplate = templateData

	// Render block id and save to block/id/{hash}
	// Render block id and save to block/latest
	// Save summary to /
}

func main() {
	// Have to make a package here...
	// HTMX templates?

	// ctx, cancel := context.WithCancel(context.Background())
	// defer cancel()

	mux := http.NewServeMux()
	fcache := filecache.NewDefaultCache()
	fcache.Start()
	defer fcache.Stop()

	os.MkdirAll("renders/block/id", 0777)

	mux.HandleFunc("/", func(res http.ResponseWriter, req *http.Request) {
		url := req.URL.String()

		if len(url) > 0 && strings.Count(url, "..") == 0 {
			path := ""

			if len(url) == 1 {
				// Give summary
				path = renderDir + "/" + "summary.html"
			} else {
				path = renderDir + url + ".html"
			}

			data, err := fcache.ReadFile(path)
			if err != nil {
				// Error page?
				res.Write([]byte(err.Error()))
			} else {
				res.Write(data)
			}
		} else {
			// Error page wtf!!
		}
	})

	go func() {
		hashLast := ""
		t := time.NewTicker(time.Millisecond * 400)
		for {
			select {
			case <-t.C:
				response, err := http.Get("http://" + rpcUrl + "/block")
				if err != nil {
					panic(err)
				}
				data, err := io.ReadAll(response.Body)

				hashNew, err := json.GetString(data, "result", "block_id", "hash")

				if err != nil {
					panic(err)
				} else {
					if hashNew != hashLast {
						newBlock(data)
						hashLast = hashNew
					}
				}

			}
		}
	}()

	http.ListenAndServe("127.0.0.1:9999", mux)
}
