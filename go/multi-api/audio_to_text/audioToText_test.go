/*
   Copyright 2010-2020 Amazon.com, Inc. or its affiliates. All Rights Reserved.

   This file is licensed under the Apache License, Version 2.0 (the "License").
   You may not use this file except in compliance with the License. A copy of
   the License is located at

    http://aws.amazon.com/apache2.0/

   This file is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR
   CONDITIONS OF ANY KIND, either express or implied. See the License for the
   specific language governing permissions and limitations under the License.
*/
package main

import (
	"io/ioutil"
	"log"
	"os"
	"strconv"
	"strings"
	"testing"
	"time"

	guuid "github.com/google/uuid"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
)

func multiplyDuration(factor int64, d time.Duration) time.Duration {
	return time.Duration(factor) * d
}

func TestAudioToText(t *testing.T) {
	// Make sure we have all of the names,
	// but if we don't create a unique name for most
	// Get log filename from environment variable LOG_FILE
	logFile := os.Getenv("LOG_FILE")

	if logFile == "" {
		// Create a log file based on the time and date
		t := time.Now()
		logFile = "audiototext-" + t.Format("20060102150405") + "-log"

		err := os.Setenv("LOG_FILE", logFile)
		if err != nil {
			os.Exit(1)
		}
	}

	f, err := os.OpenFile(logFile,
		os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		log.Println(err)
		os.Exit(1)
	}
	defer f.Close()

	logger := log.New(f, "AUDIO_TO_TEXT", log.LstdFlags)

	// Get region from environment variable AWS_REGION
	region := os.Getenv("AWS_REGION")

	if region == "" {
		// If no region is set, use us-west-2
		region = "us-west-2"
		err = os.Setenv("AWS_REGION", region)
		if err != nil {
			logger.Println("Got an error setting the AWS_REGION env variable to us-west-2")
			os.Exit(1)
		}

		logger.Println("Setting AWS_REGION env variable to us-west-2")
	}

	/*
		Bucket names can be up to 63 characters long,
		and can contain only lower-case characters, numbers, periods, and dashes.
	*/

	// Get input bucket name from INPUT_BUCKET env variable
	inputBucket := os.Getenv("INPUT_BUCKET")

	if inputBucket == "" {
		// Create unique bucket name
		// GUIDs are 32 chars long
		id := guuid.New()
		bucketName := "input-" + id.String()
		logger.Println("Setting the INPUT_BUCKET env variable to " + bucketName)
		err = os.Setenv("INPUT_BUCKET", bucketName)
		if err != nil {
			logger.Println("Got an error setting the INPUT_BUCKET env variable to " + bucketName)
			os.Exit(1)
		}
	}

	// Get output bucket name from OUTPUT_BUCKET env variable
	outputBucket := os.Getenv("OUTPUT_BUCKET")

	if outputBucket == "" {
		// Create unique bucket name
		// GUIDs are 32 chars long
		id := guuid.New()
		bucketName := "output-" + id.String()
		logger.Println("Setting the OUTPUT_BUCKET env variable to " + bucketName)
		err = os.Setenv("OUTPUT_BUCKET", bucketName)
		if err != nil {
			logger.Println("Got an error setting the OUTPUT_BUCKET env variable to " + bucketName)
			os.Exit(1)
		}
	}

	// Get Lambda function name from LAMBDA_FUNCTION env variable
	lambdaFunction := os.Getenv("LAMBDA_FUNCTION")

	if lambdaFunction == "" {
		logger.Println("You must specify a Lambda function name in the LAMBDA_FUNCTION env variable")
		os.Exit(1)
	}

	// Get audio file name from AUDIO_FILE env variable
	audioFile := os.Getenv("AUDIO_FILE")

	if audioFile == "" {
		audioFile = "FourScore.wav"
		logger.Println("Setting the AUDIO_FILE env variable to " + audioFile)
		err = os.Setenv("AUDIO_FILE", audioFile)
		if err != nil {
			logger.Println("Got an error setting the AUDIO_FILE env variable to " + audioFile)
			os.Exit(1)
		}
	}

	// Get file to validate against from VALID_FILE env variable
	validateFile := os.Getenv("VALID_FILE")

	if validateFile == "" {
		validateFile = "FourScoreResult.txt"
		logger.Println("Setting the VALID_FILE env variable to " + validateFile)
		err = os.Setenv("VALID_FILE", validateFile)
		if err != nil {
			logger.Println("Got an error setting the VALID_FILE env variable to " + validateFile)
			os.Exit(1)
		}
	}

	// Get how long to wait, in seconds, for file to show up in output bucket from SLEEP_SECONDS env variable
	sleepSecondsStr := os.Getenv("SLEEP_SECONDS")

	if sleepSecondsStr == "" {
		sleepSecondsStr = "10"
		logger.Println("Setting the SLEEP_SECONDS env variable to " + sleepSecondsStr)
		err = os.Setenv("SLEEP_SECONDS", sleepSecondsStr)
		if err != nil {
			logger.Println("Got an error setting the SLEEP_SECONDS env variable to " + sleepSecondsStr)
			os.Exit(1)
		}
	}

	sleepSeconds, err := strconv.ParseInt(sleepSecondsStr, 10, 64)
	if err != nil {
		logger.Println(sleepSecondsStr + " is not an integer")
		os.Exit(1)
	}

	// Create a session, in the specified region, to use for all operations,
	// using default credentials
	sess := session.Must(session.NewSession(&aws.Config{
		Region: aws.String(region),
	}))

	// Upload audio file to bucket
	err = dropFile(sess, inputBucket, audioFile)
	if err != nil {
		logger.Println("Failed to drop " + audioFile + " into bucket " + inputBucket)
		os.Exit(1)
	}

	// Output file should be the same name as input file, but with a txt extension
	parts := strings.Split(inputBucket, ".")
	filename := parts[0] + ".txt"

	// Wait for sleepSeconds seconds (is there a better way???)
	ts := multiplyDuration(sleepSeconds, time.Second)
	time.Sleep(ts)

	// Get txt file from bucket
	file, err := getFile(sess, outputBucket, filename)
	if err != nil {
		logger.Println("Failed to get " + filename + " from bucket " + outputBucket)
		os.Exit(1)
	}
	defer file.Close()

	content, err := ioutil.ReadAll(file)
	gotString := string(content)

	// Compare it with FourScoreResult.txt
	validFile, err := os.Open(validateFile)
	if err != nil {
		logger.Println("Got error opening " + validateFile)
		os.Exit(1)
	}
	defer validFile.Close()

	b, err := ioutil.ReadAll(validFile)
	if err != nil {
		logger.Println("Got error reading " + validateFile)
		os.Exit(1)
	}

	cmpString := string(b)

	if gotString != cmpString {
		logger.Println("Received file content:")
		logger.Println(content)
		logger.Println("is not:")
		logger.Println(b)
	}
}
