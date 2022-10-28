package main

import (
	"context"
	"encoding/json"
	"github.com/JamesPEarly/loggly"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/feature/dynamodb/attributevalue"
	"github.com/aws/aws-sdk-go-v2/feature/dynamodb/expression"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/gorilla/mux"
	"github.com/joho/godotenv"
	"net/http"
	"regexp"
	"sort"
)

type Post struct {
	Title       string  `dynamodbav:"title"`
	FullID      string  `dynamodbav:"name"`
	Author      string  `dynamodbav:"author"`
	Permalink   string  `dynamodbav:"permalink"`
	URL         string  `dynamodbav:"url"`
	DateCreated float64 `dynamodbav:"created_utc"`
}

type Status struct {
	Table       string `json:"table"`
	RecordCount int32  `json:"recordCount"`
}

type StatusLog struct {
	Method      string `json:"method"`
	SourceIP    string `json:"ip"`
	RequestPath string `json:"path"`
	StatusCode  int    `json:"status"`
}

var logger *loggly.ClientType
var db *dynamodb.Client

func init() {
	_ = godotenv.Load()

	logger = loggly.New("rnelson3-server")
	cfg, _ := config.LoadDefaultConfig(context.TODO())
	db = dynamodb.NewFromConfig(cfg)
}

func main() {
	router := mux.NewRouter()

	router.HandleFunc("/rnelson3/all", allHandler).Methods("GET")
	router.HandleFunc("/rnelson3/status", statusHandler).Methods("GET")
	router.HandleFunc("/rnelson3/search", searchHandler).Methods("GET")
	router.Methods("POST", "PUT", "PATCH", "DELETE").HandlerFunc(methodNotAllowedHandler)
	router.PathPrefix("/rnelson3/").HandlerFunc(notFoundHandler)

	_ = logger.EchoSend("info", "Ready!")
	_ = http.ListenAndServe(":8080", router)
}

func allHandler(res http.ResponseWriter, req *http.Request) {
	output, _ := db.Scan(context.TODO(), &dynamodb.ScanInput{TableName: aws.String("rnelson3-reddit")})

	var posts []Post

	for _, item := range output.Items {
		var post Post
		_ = attributevalue.UnmarshalMap(item, &post)
		posts = append(posts, post)
	}

	sort.Slice(posts, func(i, j int) bool {
		return posts[i].DateCreated > posts[j].DateCreated
	})

	data, _ := json.MarshalIndent(posts, "", "    ")

	res.Header().Add("Content-Type", "application/json")
	res.WriteHeader(http.StatusOK)
	_, _ = res.Write(data)

	logRequest(req, http.StatusOK)
}

func statusHandler(res http.ResponseWriter, req *http.Request) {
	output, _ := db.Scan(context.TODO(), &dynamodb.ScanInput{TableName: aws.String("rnelson3-reddit")})

	status := Status{
		Table:       "rnelson3-reddit",
		RecordCount: output.Count,
	}

	data, _ := json.MarshalIndent(status, "", "    ")

	res.Header().Add("Content-Type", "application/json")
	res.WriteHeader(http.StatusOK)
	_, _ = res.Write(data)

	logRequest(req, http.StatusOK)
}

func searchHandler(res http.ResponseWriter, req *http.Request) {
	params := req.URL.Query()
	var posts []Post

	for key, value := range params {
		match := true

		switch key {
		case "name":
			match, _ = regexp.MatchString("t3_[a-z0-9]{6}", value[0])
			break

		case "author":
			match, _ = regexp.MatchString("[A-Za-z0-9-_]+", value[0])
			break

		case "created_utc":
			match, _ = regexp.MatchString("\\d{10}", value[0])
			break

		case "permalink":
			match, _ = regexp.MatchString("/r/FloridaMan/comments/t3_[a-z0-9]{6}/.+", value[0])
			break

		case "title":
			match, _ = regexp.MatchString(".+", value[0])
			break

		case "url":
			match, _ = regexp.MatchString("https://.+", value[0])
			break

		default:
			match = false
			break
		}

		if !match {
			res.WriteHeader(http.StatusBadRequest)
			_, _ = res.Write([]byte("400 Bad Request"))
			logRequest(req, http.StatusBadRequest)

			return
		}

		filter := expression.Name(key).Equal(expression.Value(value[0]))
		proj := expression.NamesList(expression.Name(key))
		expr, _ := expression.NewBuilder().WithFilter(filter).WithProjection(proj).Build()

		output, _ := db.Scan(context.TODO(), &dynamodb.ScanInput{
			TableName:                 aws.String("rnelson3-reddit"),
			ExpressionAttributeNames:  expr.Names(),
			ExpressionAttributeValues: expr.Values(),
			FilterExpression:          expr.Filter(),
		})

		_ = attributevalue.UnmarshalListOfMaps(output.Items, &posts)

		break
	}

	data, _ := json.MarshalIndent(posts, "", "    ")

	res.Header().Add("Content-Type", "application/json")
	res.WriteHeader(http.StatusOK)
	_, _ = res.Write(data)

	logRequest(req, http.StatusOK)
}

func methodNotAllowedHandler(res http.ResponseWriter, req *http.Request) {
	res.WriteHeader(http.StatusMethodNotAllowed)
	_, _ = res.Write([]byte("405 Method Not Allowed"))
	logRequest(req, http.StatusMethodNotAllowed)
}

func notFoundHandler(res http.ResponseWriter, req *http.Request) {
	res.WriteHeader(http.StatusNotFound)
	_, _ = res.Write([]byte("404 Not Found"))
	logRequest(req, http.StatusNotFound)
}

func logRequest(req *http.Request, statusCode int) {
	bytes, _ := json.MarshalIndent(StatusLog{
		Method:      req.Method,
		SourceIP:    req.RemoteAddr,
		RequestPath: req.URL.Path,
		StatusCode:  statusCode,
	}, "", "    ")

	_ = logger.EchoSend("info", string(bytes))
}
