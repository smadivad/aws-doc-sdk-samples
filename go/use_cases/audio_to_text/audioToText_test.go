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
    "encoding/json"
    "io/ioutil"
    "os"
    "strconv"
    "testing"
    "time"

    "github.com/aws/aws-sdk-go/aws"
    "github.com/aws/aws-sdk-go/aws/session"
    "github.com/aws/aws-sdk-go/service/s3"
    "github.com/aws/aws-sdk-go/service/s3/s3manager"

    "github.com/google/uuid"
)

type Config struct {
    InputBucket  string `json:"InputBucket"`
    OutputBucket string `json:"OutputBucket"`
    AudioFile    string `json:"AudioFile"`
    ResultFile   string `json:"ResultFile"`
    SleepSeconds int    `json:"SleepSeconds"`
    JobName      string `json:"JobName"`
    Debug        bool   `json:"Debug"`
}

func createBucket(sess *session.Session, bucket *string) error {
    svc := s3.New(sess)

    _, err := svc.CreateBucket(&s3.CreateBucketInput{
        Bucket: bucket,
    })
    if err != nil {
        return err
    }

    err = svc.WaitUntilBucketExists(&s3.HeadBucketInput{
        Bucket: bucket,
    })
    if err != nil {
        return err
    }

    return nil
}

func deleteBucket(sess *session.Session, bucket *string) error {
    svc := s3.New(sess)

    iter := s3manager.NewDeleteListIterator(svc, &s3.ListObjectsInput{
        Bucket: bucket,
    })

    err := s3manager.NewBatchDeleteWithClient(svc).Delete(aws.BackgroundContext(), iter)
    if err != nil {
        return err
    }

    _, err = svc.DeleteBucket(&s3.DeleteBucketInput{
        Bucket: bucket,
    })
    if err != nil {
        return err
    }

    err = svc.WaitUntilBucketNotExists(&s3.HeadBucketInput{
        Bucket: bucket,
    })
    if err != nil {
        return err
    }

    return nil
}

func cleanUp(t *testing.T, sess *session.Session, inputBucket, outputBucket *string, inputBucketCreated, outputBuckeCreated bool) {
    if inputBucketCreated {
        err := deleteBucket(sess, inputBucket)
        if err != nil {
            t.Log("You'll have to delete input bucket " + *inputBucket + " yourself")
        }

        t.Log("Deleted input bucket " + *inputBucket)
    }

    if outputBuckeCreated {
        err := deleteBucket(sess, outputBucket)
        if err != nil {
            t.Log("You'll have to delete output bucket " + *outputBucket + " yourself")
        }

        t.Log("Deleted output bucket " + *outputBucket)
    }
}

func getName(configName, envName, defaultName string) (bool, string) {
    if configName != "" {
        return false, configName
    }

    name := os.Getenv(envName)

    if name != "" {
        return false, name
    }

    return true, defaultName
}

func TestAudioToText(t *testing.T) {
    // When the test started:
    thisTime := time.Now()
    nowString := thisTime.Format("20060102150405")
    t.Log("Starting unit test at " + nowString)

    // Get configuration from config.json
    configFileName := "config.json"

    // Get entire file as a JSON string
    content, err := ioutil.ReadFile(configFileName)
    if err != nil {
        t.Fatal(err)
    }

    // Convert []byte to string
    text := string(content)
    config := Config{}

    // Marshall JSON string in text into job struct
    err = json.Unmarshal([]byte(text), &config)
    if err != nil {
        t.Fatal(err)
    }

    // Add timestamp to job name
    config.JobName = config.JobName + "-" + nowString

    t.Log("InputBucket:  " + config.InputBucket)
    t.Log("OutputBucket: " + config.OutputBucket)
    t.Log("AudioFile:    " + config.AudioFile)
    t.Log("ResultFile:   " + config.ResultFile)
    seconds := strconv.Itoa(config.SleepSeconds)
    t.Log("SleepSeconds: " + seconds)
    t.Log("JobName:      " + config.JobName)
    if config.Debug {
        t.Log("Debug:        enabled")
    } else {
        t.Log("Debug:        disabled")
    }

    // Create random string
    id := uuid.New()
    defaultName := id.String()

    _, audioFile := getName(config.AudioFile, "AUDIO_FILE", "")
    _, resultsFile := getName(config.ResultFile, "RESULTS_FILE", "")

    if audioFile == "" || resultsFile == "" {
        t.Fatal("You must supply the name of the audio file in config.json or AUDIO_FILE env variable and the name of the results file in config.json or RESULTS_FILE env variable.")
    }

    // Get input bucket name
    inputBucketCreated, inputBucket := getName(config.InputBucket, "INPUT_BUCKET", "input-"+defaultName)
    outputBucketCreated, outputBucket := getName(config.OutputBucket, "OUTPUT_BUCKET", "output-"+defaultName)

    _, sleepSecondsStr := getName(seconds, "SLEEP_SECONDS", "")

    sleepSeconds, err := strconv.ParseInt(sleepSecondsStr, 10, 64)
    if err != nil {
        t.Fatal(err)
    }

    _, jobName := getName(config.JobName, "JOB_NAME", "ConvertAudioToText")
    jobName = jobName + nowString

    sess := session.Must(session.NewSessionWithOptions(session.Options{
        SharedConfigState: session.SharedConfigEnable,
    }))

    if inputBucketCreated {
        err = createBucket(sess, &inputBucket)
        if err != nil {
            t.Fatal(err)
        }
        t.Log("Created input bucket " + inputBucket)
    }

    if outputBucketCreated {
        err = createBucket(sess, &outputBucket)
        if err != nil {
            t.Fatal(err)
        }
        t.Log("Created output bucket " + outputBucket)
    }

    t.Log("Running transcription, which may take a couple of minutes")

    duration, err := RunTranscription(sess, &audioFile, &inputBucket, &outputBucket, &jobName, &resultsFile, &sleepSeconds)
    if err != nil {
        cleanUp(t, sess, &inputBucket, &outputBucket, inputBucketCreated, outputBucketCreated)
        t.Fatal(err)
    }

    t.Log("Transcription was successful after " + strconv.Itoa(int(duration)) + " seconds")

    cleanUp(t, sess, &inputBucket, &outputBucket, inputBucketCreated, outputBucketCreated)
}
