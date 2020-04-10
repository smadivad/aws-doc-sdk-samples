/*
   Copyright Amazon.com, Inc. or its affiliates. All Rights Reserved.

   This file is licensed under the Apache License, Version 2.0 (the "License").
   You may not use this file except in compliance with the License. A copy of
   the License is located at

    http://aws.amazon.com/apache2.0/

   This file is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR
   CONDITIONS OF ANY KIND, either express or implied. See the License for the
   specific language governing permissions and limitations under the License.
*/
// snippet-start:[use_case.go.audio_to_text]
package main

// snippet-start:[use_case.go.audio_to_text.imports]
import (
    "bytes"
    "encoding/json"
    "errors"
    "flag"
    "fmt"
    "io/ioutil"
    "os"
    "strconv"
    "strings"
    "time"

    "github.com/aws/aws-sdk-go/aws"
    "github.com/aws/aws-sdk-go/aws/session"
    "github.com/aws/aws-sdk-go/service/s3"
    "github.com/aws/aws-sdk-go/service/s3/s3manager"
    "github.com/aws/aws-sdk-go/service/transcribeservice"
)
// snippet-end:[use_case.go.audio_to_text.imports]

// TranscriptResults defines a transcription result from JSON
type TranscriptResults struct {
    JobName   string `json:"jobName"`
    AccountID string `json:"accountId"`
    Results   struct {
        Transcripts []struct {
            Transcript string `json:"transcript"`
        } `json:"transcripts"`
        Items []struct {
            StartTime    string `json:"start_time,omitempty"`
            EndTime      string `json:"end_time,omitempty"`
            Alternatives []struct {
                Confidence string `json:"confidence"`
                Content    string `json:"content"`
            } `json:"alternatives"`
            Type string `json:"type"`
        } `json:"items"`
    } `json:"results"`
    Status string `json:"status"`
}

// dropFile adds MP3/4 file to a bucket
// and returns the URI (for StartTranscriptionJob)
// snippet-start:[use_case.go.audio_to_text.drop_file]
func dropFile(sess *session.Session, filename, inputBucket *string) (string, error) {
    file, err := os.Open(*filename)
    if err != nil {
        fmt.Println("Could not open " + *filename)
        return "", err
    }

    defer file.Close()

    uploader := s3manager.NewUploader(sess)

    // Upload the file's body to S3 bucket as an object with the key being the
    // same as the filename.
    _, err = uploader.Upload(&s3manager.UploadInput{
        Bucket: inputBucket,
        Key:    filename,
        Body:   file,
    })
    if err != nil {
        fmt.Println("Failed to upload " + *filename + " to bucket " + *inputBucket)
        return "", err
    }

    // Create URI based on bucket name and filename
    uri := "s3://" + *inputBucket + "/" + *filename

    return uri, nil
}
// snippet-end:[use_case.go.audio_to_text.drop_file]

// startTranscription starts a transcription session
// snippet-start:[use_case.go.audio_to_text.start_transcription]
func startTranscription(sess *session.Session, mediaURI, outputBucket, jobName *string) error {
    svc := transcribeservice.New(sess)

    input := &transcribeservice.StartTranscriptionJobInput{
        LanguageCode: aws.String("en-US"),
        Media: &transcribeservice.Media{
            MediaFileUri: mediaURI,
        },
        MediaFormat:          aws.String("wav"),
        OutputBucketName:     outputBucket,
        TranscriptionJobName: jobName,
    }

    _, err := svc.StartTranscriptionJob(input)
    if err != nil {
        return err
    }

    return nil
}
// snippet-end:[use_case.go.audio_to_text.start_transcription]

// isTranscriptionDone determines whether a transcription is finished.
// snippet-start:[use_case.go.audio_to_text.is_transcription_done]
func isTranscriptionDone(sess *session.Session, jobName *string) (bool, error) {
    svc := transcribeservice.New(sess)

    input := &transcribeservice.ListTranscriptionJobsInput{
        JobNameContains: jobName,
    }

    result, err := svc.ListTranscriptionJobs(input)
    if err != nil {
        return false, err
    }

    if result.TranscriptionJobSummaries[0].TranscriptionJobStatus == nil {
        return false, nil
    }

    // QUEUED | IN_PROGRESS | FAILED | COMPLETED
    if *result.TranscriptionJobSummaries[0].TranscriptionJobStatus == "COMPLETED" {
        return true, nil
    }

    if *result.TranscriptionJobSummaries[0].TranscriptionJobStatus == "FAILED" {
        return false, errors.New("Job failed")
    }

    return false, nil
}
// snippet-end:[use_case.go.audio_to_text.is_transcription_done]

// getResultURI retrieves the URI for the resulting transcription.
// snippet-start:[use_case.go.audio_to_text.get_result_uri]
func getResultURI(sess *session.Session, jobName *string) (string, error) {
    svc := transcribeservice.New(sess)

    input := &transcribeservice.GetTranscriptionJobInput{
        TranscriptionJobName: jobName,
    }

    results, err := svc.GetTranscriptionJob(input)
    if err != nil {
        return "", err
    }

    return *results.TranscriptionJob.Transcript.TranscriptFileUri, nil
}
// snippet-end:[use_case.go.audio_to_text.get_result_uri]

// getTextFromURI retrieves the text from the URI of the transcription.
// snippet-start:[use_case.go.audio_to_text.get_text_from_uri]
func getTextFromURI(sess *session.Session, outputBucket, uri *string) (string, error) {
    svc := s3.New(sess)
    // The URI should look something like:
    // "s3://" + outputBucket + "/" + filename
    parts := strings.Split(*uri, "/")
    filename := parts[len(parts)-1]

    // Now get the guts of the file and return it
    input := &s3.GetObjectInput{
        Bucket: outputBucket,
        Key:    aws.String(filename),
    }

    result, err := svc.GetObject(input)
    if err != nil {
        return "", err
    }

    // Body is of type io.ReadCloser
    buf := new(bytes.Buffer)
    _, err = buf.ReadFrom(result.Body)
    if err != nil {
        return "", err
    }

    text := buf.String()

    job := TranscriptResults{}

    // Marshall JSON string in text into job struct
    err = json.Unmarshal([]byte(text), &job)
    if err != nil {
        return "", err
    }

    return job.Results.Transcripts[0].Transcript, nil
}
// snippet-end:[use_case.go.audio_to_text.get_text_from_uri]

// getTextFromFile retrieves the contents of a file as text
// snippet-start:[use_case.go.audio_to_text.get_text_from_file]
func getTextFromFile(filename *string) (string, error) {
    file, err := os.Open(*filename)
    if err != nil {
        return "", err
    }

    defer file.Close()

    b, err := ioutil.ReadAll(file)
    if err != nil {
        return "", err
    }

    return string(b), nil
}
// snippet-end:[use_case.go.audio_to_text.get_text_from_file]

// snippet-start:[use_case.go.audio_to_text.multiply_duration]
func multiplyDuration(factor int64, d time.Duration) time.Duration {
    return time.Duration(factor) * d
}
// snippet-end:[use_case.go.audio_to_text.multiply_duration]

// RunTranscription transcribes an audio file and compares it to the expected output
// snippet-start:[use_case.go.audio_to_text.run_transcription]
func RunTranscription(sess *session.Session, audioFile, inputBucket, outputBucket, jobName, resultsFile *string, sleepSeconds *int64) (int64, error) {
    var duration int64

    // Upload audio file to bucket
    fileURI, err := dropFile(sess, audioFile, inputBucket)
    if err != nil {
        return duration, err
    }

    err = startTranscription(sess, &fileURI, outputBucket, jobName)
    if err != nil {
        return duration, err
    }

    // loop until the job is done
    // Wait for sleepSeconds seconds (is there a better way???)
    ts := multiplyDuration(*sleepSeconds, time.Second)

    for {
        done, err := isTranscriptionDone(sess, jobName)
        if err != nil {
            return duration, err
        }

        if done {
            break
        }

        time.Sleep(ts)
        duration += *sleepSeconds
    }

    // Now get results of transcription
    resultsURI, err := getResultURI(sess, jobName)
    if err != nil {
        return duration, err
    }

    text, err := getTextFromURI(sess, outputBucket, &resultsURI)
    if err != nil {
        return duration, err
    }

    expected, err := getTextFromFile(resultsFile)
    if err != nil {
        return duration, err
    }

    // Compare the results with the expected results
    if text != expected {
        msg := "Did NOT get the expected results. Got:\n" + text + "' instead of: '" + expected + "'"
        return duration, errors.New(msg)
    }

    return duration, nil
}
// snippet-end:[use_case.go.audio_to_text.run_transcription]

func main() {
    // snippet-start:[use_case.go.audio_to_text.args]
    inputBucket := flag.String("i", "", "The input bucket")
    outputBucket := flag.String("o", "", "The output bucket")
    audioFile := flag.String("a", "FourScore.wav", "The file containing the audio")
    resultFile := flag.String("f", "FourScoreResult.txt", "File containing the confirming results")
    sleepSeconds := flag.Int64("s", 10, "How long to sleep between checking whether transcription is complete")
    jobName := flag.String("j", "ConvertAudioToText", "The name of the transcription job")

    if *inputBucket == "" || *outputBucket == "" || *audioFile == "" || *resultFile == "" || *sleepSeconds < 0 || *jobName == "" {
        fmt.Println("You must supply a value for region, input bucket, output bucket, audio file, result file, and job name. Also, a sleep value must be > 0")
        return
    }
    // snippet-end:[use_case.go.audio_to_text.args]

    // snippet-start:[use_case.go.audio_to_text.session]
    sess := session.Must(session.NewSessionWithOptions(session.Options{
        SharedConfigState: session.SharedConfigEnable,
    }))
    // snippet-end:[use_case.go.audio_to_text.session]

    // snippet-start:[use_case.go.audio_to_text.call]
    duration, err := RunTranscription(sess, audioFile, inputBucket, outputBucket, jobName, resultFile, sleepSeconds)
    if err != nil {
        fmt.Println("Transcription failed after " + strconv.Itoa(int(duration)) + " seconds with error:")
        fmt.Println(err)
        return
    }

    fmt.Println("Transcription was successful after waiting " + strconv.Itoa(int(duration)) + " seconds")
    // snippet-end:[use_case.go.audio_to_text.call]
}
// snippet-end:[use_case.go.audio_to_text]
