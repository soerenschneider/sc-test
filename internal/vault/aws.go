package vault

import (
	"cmp"
	"context"
	"os"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/sts"
)

const awsDefaultRegion = "us-east-1"

// WaitForAwsCredentials polls sts:GetCallerIdentity until AWS accepts the
// credentials. Freshly created IAM access keys are eventually consistent and
// get rejected for a few seconds after Vault hands them out. It returns the ARN
// of the caller, or the last error once ctx is done.
func WaitForAwsCredentials(ctx context.Context, creds AwsCredentials, interval time.Duration) (string, error) {
	client := sts.New(sts.Options{
		Region:      cmp.Or(os.Getenv("AWS_REGION"), os.Getenv("AWS_DEFAULT_REGION"), awsDefaultRegion),
		Credentials: credentials.NewStaticCredentialsProvider(creds.AccessKeyID, creds.SecretAccessKey, creds.SessionToken),
		// Rejected credentials are the expected case here, retrying is done below.
		RetryMaxAttempts: 1,
	})

	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	var lastErr error
	for {
		out, err := client.GetCallerIdentity(ctx, &sts.GetCallerIdentityInput{})
		if err == nil {
			return aws.ToString(out.Arn), nil
		}
		// Prefer the error AWS returned over the cancelled context's one.
		if ctx.Err() == nil || lastErr == nil {
			lastErr = err
		}

		select {
		case <-ctx.Done():
			return "", lastErr
		case <-ticker.C:
		}
	}
}
