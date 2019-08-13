package s3

import (
	"testing"

	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/stretchr/testify/assert"
)

func setEnv() {
	os.Setenv("AWS_ACCESS_KEY_ID", "accessKey")
	os.Setenv("AWS_SECRET_ACCESS_KEY", "secret")
}

func clearnEnv() {
	os.Unsetenv("AWS_ACCESS_KEY_ID")
	os.Unsetenv("AWS_SECRET_ACCESS_KEY")
	os.Unsetenv("AWS_SHARED_CREDENTIALS_FILE")
	os.Unsetenv("AWS_CONFIG_FILE")
}

func Test_awsSession(t *testing.T) {
	type args struct {
		profile string
	}

	tests := []struct {
		name             string
		args             args
		want             *session.Session
		before           func()
		expectToken      string
		endpoint         string
		S3ForcePathStyle bool
	}{
		{
			name:     "test endpoint",
			before:   setEnv,
			endpoint: "test",
		},
		{
			name:             "test S3ForcePathStyle",
			before:           setEnv,
			S3ForcePathStyle: true,
		},
		{
			name: "test env provider",
			args: args{
				profile: "test1",
			},
			before: setEnv,
		},
		{
			name: "test default shared credentials provider",
			before: func() {
				clearnEnv()
				os.Setenv("AWS_SHARED_CREDENTIALS_FILE", filepath.Join("testdata", "credentials.ini"))
			},
			expectToken: "token",
		},
		{
			name: "test default shared credentials provider",
			before: func() {
				clearnEnv()
				os.Setenv("AWS_SHARED_CREDENTIALS_FILE", filepath.Join("testdata", "credentials.ini"))
			},
			expectToken: "token",
		},
		{
			name: "test profile with shared credentials provider",
			before: func() {
				clearnEnv()
				os.Setenv("AWS_SHARED_CREDENTIALS_FILE", filepath.Join("testdata", "credentials.ini"))
			},
			args: args{
				profile: "no_token",
			},
			expectToken: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			clearnEnv()
			defer clearnEnv()
			if tt.before != nil {
				tt.before()
			}

			builder := newSessionBuilder()
			builder.Profile(tt.args.profile)
			builder.Endpoint(tt.endpoint)
			builder.S3ForcePathStyle(tt.S3ForcePathStyle)
			sess := builder.Build()
			assert.NotNil(t, sess)

			creds, err := sess.Config.Credentials.Get()
			assert.Nil(t, err)

			assert.Equal(t, "accessKey", creds.AccessKeyID, "Expect access key ID to match")
			assert.Equal(t, "secret", creds.SecretAccessKey, "Expect secret access key to match")
			assert.Equal(t, tt.expectToken, creds.SessionToken, "Expect token to match")

			assert.Equal(t, aws.String(tt.endpoint), sess.Config.Endpoint, "Expect endpoint to match")
			assert.Equal(t, aws.Bool(tt.S3ForcePathStyle), sess.Config.S3ForcePathStyle, "Expect S3ForcePathStyle to match")
		})
	}
}

const assumeRoleRespMsg = `
<AssumeRoleResponse xmlns="https://sts.amazonaws.com/doc/2011-06-15/">
  <AssumeRoleResult>
    <AssumedRoleUser>
      <Arn>arn:aws:sts::account_id:assumed-role/role/session_name</Arn>
      <AssumedRoleId>AKID:session_name</AssumedRoleId>
    </AssumedRoleUser>
    <Credentials>
      <AccessKeyId>AKID</AccessKeyId>
      <SecretAccessKey>SECRET</SecretAccessKey>
      <SessionToken>SESSION_TOKEN</SessionToken>
      <Expiration>%s</Expiration>
    </Credentials>
  </AssumeRoleResult>
  <ResponseMetadata>
    <RequestId>request-id</RequestId>
  </ResponseMetadata>
</AssumeRoleResponse>
`

func Test_awsSession_mfa(t *testing.T) {
	clearnEnv()
	defer clearnEnv()
	os.Setenv("AWS_SHARED_CREDENTIALS_FILE", filepath.Join("testdata", "credentials.ini"))
	os.Setenv("AWS_CONFIG_FILE", filepath.Join("testdata", "config.ini"))

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, r.FormValue("SerialNumber"), "arn:aws:iam::1111111111:mfa/test")
		assert.Equal(t, r.FormValue("TokenCode"), "tokencode")

		_, err := w.Write([]byte(fmt.Sprintf(assumeRoleRespMsg, time.Now().Add(15*time.Minute).Format("2006-01-02T15:04:05Z"))))
		assert.NoError(t, err)
	}))

	customProviderCalled := false

	options := &session.Options{
		Profile: "cloudformation@flowlab-dev",
		Config: aws.Config{
			Region:     aws.String("eu-west-1"),
			Endpoint:   aws.String(server.URL),
			DisableSSL: aws.Bool(true),
		},
		SharedConfigState: session.SharedConfigEnable,
		AssumeRoleTokenProvider: func() (string, error) {
			customProviderCalled = true
			return "tokencode", nil
		},
	}

	builder := newSessionBuilder()
	builder.Profile("cloudformation@flowlab-dev")
	builder.Options(options)
	sess := builder.Build()

	creds, err := sess.Config.Credentials.Get()
	assert.NoError(t, err)
	assert.True(t, customProviderCalled)
	assert.Contains(t, creds.ProviderName, "AssumeRoleProvider")
}

func Test_awsSession_fail(t *testing.T) {
	tests := []struct {
		name string
	}{
		{
			name: "should fail with no credentials",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			clearnEnv()
			defer clearnEnv()
			os.Setenv("AWS_SHARED_CREDENTIALS_FILE", "/nope")

			builder := newSessionBuilder()
			sess := builder.Build()
			assert.NotNil(t, sess)

			_, err := sess.Config.Credentials.Get()
			assert.NotNil(t, err)
		})
	}
}
