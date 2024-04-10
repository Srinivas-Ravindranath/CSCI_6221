/*
Package for handling the requests sent to the API gateway and call
the correct functions to handle the request
TODO: Simply the initialize process by removing re-use of initialize function
*/
package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"regexp"
	subscriptions "subHandler/src/subscription_handler"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/dynamodb"
	"github.com/google/uuid"
)

type dynamoAttr struct {
	dynamoCli *dynamodb.DynamoDB
	awsRegion string
	tableName string
}

type GetItem struct {
	UserName string
}

type HandlerFunc func(context.Context, events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error)

func initialize() dynamoAttr {
	/*
		Used to initialize the table attributes and sdk clients
		Params: None
		Return: None
	*/
	awsRegion := os.Getenv("aws_region") // Get from ENV vars of lambda
	dynamodbTable := os.Getenv("table_name")

	sess, err := session.NewSession(&aws.Config{
		Region: aws.String(awsRegion),
	})

	if err != nil {
		fmt.Println("Error creating sess", err)
		os.Exit(0)
	}

	dynamoClient := dynamodb.New(sess)
	return dynamoAttr{
		dynamoCli: dynamoClient,
		awsRegion: awsRegion,
		tableName: dynamodbTable,
	}
}

func handlerSubscription(ctx context.Context, request events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
	/*
		Handles the dynamodb Put and Get operation for the API gateway calls
		GET request returns all the subscriptions based on the username
		POST adds the given subscription to the table
		Params: ctx context.Context
				request events.APIGatewayProxyRequest
		Returns: events.APIGatewayProxyResponse
		         error
	*/

	httpMethod := request.HTTPMethod

	dynamoCli := initialize()
	if httpMethod == "GET" {

		userName := request.PathParameters["userName"]

		subscriptionsResponse := subscriptions.GetSubscriptions(dynamoCli.dynamoCli, dynamoCli.tableName, userName)
		responseBody, err := json.Marshal(subscriptionsResponse)
		if err != nil {
			return events.APIGatewayProxyResponse{StatusCode: 500}, err
		}
		return events.APIGatewayProxyResponse{
			StatusCode: subscriptionsResponse.StatusCode,
			Body:       string(responseBody),
		}, nil
	}
	if httpMethod == "POST" {
		var addItem subscriptions.AddItem
		err := json.Unmarshal([]byte(request.Body), &addItem)
		finalAddItem := subscriptions.AddItem{
			UUID:                 uuid.New().String(),
			UserName:             addItem.UserName,
			VendorName:           addItem.VendorName,
			VendorUrl:            addItem.VendorUrl,
			SubscriptionDuration: addItem.SubscriptionDuration,
			RemindTime:           subscriptions.CalcRemindTime(addItem.SubscriptionDuration), // need to use this to calc RemindTime
		}
		if err != nil {
			return events.APIGatewayProxyResponse{StatusCode: 400, Body: "Bad Request"}, nil
		}
		subscriptionsPostResponse := subscriptions.AddItemToTable(dynamoCli.dynamoCli, dynamoCli.tableName, finalAddItem)
		responseBody, err := json.Marshal(subscriptionsPostResponse)
		if err != nil {
			return events.APIGatewayProxyResponse{StatusCode: 500}, err
		}
		return events.APIGatewayProxyResponse{
			StatusCode: subscriptionsPostResponse.StatusCode,
			Body:       string(responseBody),
		}, nil
	}
	return events.APIGatewayProxyResponse{StatusCode: 400, Body: "Bad Request"}, nil
}

func handlerSubscriptionID(ctx context.Context, request events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
	/*
		Handles the dynamodb Get and Delete operation for the API gateway calls
		GET request returns information of a specific ID and username
		DELETE request is used to remove the specified Item from the table
		Params: ctx context.Context
				request events.APIGatewayProxyRequest
		Returns: events.APIGatewayProxyResponse
		         error
	*/
	httpMethod := request.HTTPMethod
	dynamoCli := initialize()
	if httpMethod == "GET" {
		subscriptionID := request.PathParameters["subscription_id"]
		userName := request.PathParameters["userName"]
		subscriptionResponse := subscriptions.GetSubscription(dynamoCli.dynamoCli, dynamoCli.tableName, subscriptionID, userName)
		responseBody, err := json.Marshal(subscriptionResponse)
		if err != nil {
			return events.APIGatewayProxyResponse{StatusCode: 500}, err
		}
		return events.APIGatewayProxyResponse{
			StatusCode: subscriptionResponse.Status,
			Body:       string(responseBody),
		}, nil
	}
	if httpMethod == "DELETE" {
		subscriptionID := request.PathParameters["subscription_id"]
		userName := request.PathParameters["userName"]
		deleteResponse := subscriptions.DeleteItemFromTable(dynamoCli.dynamoCli, dynamoCli.tableName, subscriptionID, userName)
		responseBody, err := json.Marshal(deleteResponse)
		if err != nil {
			return events.APIGatewayProxyResponse{StatusCode: 500}, err
		}
		return events.APIGatewayProxyResponse{
			StatusCode: deleteResponse.StatusCode,
			Body:       string(responseBody),
		}, nil
	}
	return events.APIGatewayProxyResponse{StatusCode: 400, Body: "Bad Request"}, nil
}

func handlerUpdate(ctx context.Context, request events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
	/*
		Handles the dynamodb UPDATE operation for the API gateway calls
		POST updates the given subscription attributes of the table
		Params: ctx context.Context
				request events.APIGatewayProxyRequest
		Returns: events.APIGatewayProxyResponse
		         error
	*/
	httpMethod := request.HTTPMethod
	dynamoCli := initialize()
	if httpMethod == "POST" {
		var updateItem subscriptions.UpdateItem
		subscriptionID := request.PathParameters["subscription_id"]
		err := json.Unmarshal([]byte(request.Body), &updateItem)
		finalUpdateItem := subscriptions.UpdateItem{
			UserName:             updateItem.UserName,
			VendorName:           updateItem.VendorName,
			VendorUrl:            updateItem.VendorUrl,
			SubscriptionDuration: updateItem.SubscriptionDuration,
			RemindTime:           subscriptions.CalcRemindTime(updateItem.SubscriptionDuration),
		}
		if err != nil {
			return events.APIGatewayProxyResponse{StatusCode: 400, Body: "Bad Request"}, nil
		}
		updateResponse := subscriptions.UpdateSubscription(dynamoCli.dynamoCli, dynamoCli.tableName, subscriptionID, finalUpdateItem)
		responseBody, err := json.Marshal(updateResponse)
		if err != nil {
			return events.APIGatewayProxyResponse{StatusCode: 500}, err
		}
		return events.APIGatewayProxyResponse{
			StatusCode: updateResponse.Status,
			Body:       string(responseBody),
		}, nil
	}
	return events.APIGatewayProxyResponse{StatusCode: 400, Body: "Bad Request"}, nil
}

func callHandler(hfunc HandlerFunc, ctx context.Context, request events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
	/*
		Handles the routing of the specific API call to the respective function.
		Also, handles the response and error handling for the API calls.
		Params: hfunc A function of type HandlerFunc which is the handler function to be called.
				ctx context.Context
				request
		Returns: events.APIGatewayProxyResponse
				 error
	*/
	response, err := hfunc(ctx, request)
	if err != nil {
		return events.APIGatewayProxyResponse{StatusCode: 500, Body: "Internal Server Error"}, err
	}

	responseJSON, err := json.Marshal(response.Body)
	if err != nil {
		return events.APIGatewayProxyResponse{StatusCode: 500, Body: "Internal Server Error"}, err
	}

	return events.APIGatewayProxyResponse{
		StatusCode: response.StatusCode,
		Body:       string(responseJSON),
		Headers: map[string]string{
			"Access-Control-Allow-Origin":      "*",
			"Access-Control-Allow-Methods":     "GET, POST, OPTIONS, DELETE",
			"Access-Control-Allow-Headers":     "Content-Type, Authorization",
			"Access-Control-Allow-Credentials": "true",
		},
	}, nil
}

func handlerPath(ctx context.Context, request events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
	/*
		Handles the routing of the specific API call to the respective function
		Params: ctx context.Context
				request events.APIGatewayProxyRequest
		Returns: events.APIGatewayProxyResponse
		         error
	*/
	path := request.Path
	var handlerFunc HandlerFunc
	fmt.Println("Entering Handler path: ", path)

	// We need to regex match the path after subscription/* to parse the url correctly
	subscriptionRegex, err := regexp.Compile(`^\/subscriptions$`)
	if err != nil {
		return events.APIGatewayProxyResponse{StatusCode: 500, Body: "Internal Server Error"}, err
	}

	subscriptionIDRegex, err := regexp.Compile(`^\/subscriptions\/([0-9a-zA-Z-]+)\/user\/([0-9a-zA-Z-]+)$`)
	if err != nil {
		return events.APIGatewayProxyResponse{StatusCode: 500, Body: "Internal Server Error"}, err
	}

	subscriptionListRegex, err := regexp.Compile(`^\/subscriptions\/list\/([0-9a-zA-Z-]+)$`)
	if err != nil {
		return events.APIGatewayProxyResponse{StatusCode: 500, Body: "Internal Server Error"}, err
	}

	// We need to regex match the path after subscription/update/* to parse the url correctly
	updateSubscriptionIDRegex, err := regexp.Compile(`^\/subscriptions\/update\/([0-9a-zA-Z-]+)$`)
	if err != nil {
		return events.APIGatewayProxyResponse{StatusCode: 500, Body: "Internal Server Error"}, err
	}

	// Call the correct function only when regex matches AKA results in BOOL True
	switch true {
	case subscriptionListRegex.MatchString(path) || subscriptionRegex.MatchString(path):
		handlerFunc = handlerSubscription
	case subscriptionIDRegex.MatchString(path):
		handlerFunc = handlerSubscriptionID
	case updateSubscriptionIDRegex.MatchString(path):
		handlerFunc = handlerUpdate
	default:
		return events.APIGatewayProxyResponse{StatusCode: 404, Body: "Not Found"}, nil
	}

	res, err := callHandler(handlerFunc, ctx, request)
	if err != nil {
		return events.APIGatewayProxyResponse{StatusCode: 500, Body: "Internal Server Error"}, err
	}
	return res, nil
}

func main() {
	initialize()

	lambda.Start(handlerPath)
}