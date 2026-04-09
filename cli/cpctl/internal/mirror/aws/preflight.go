package aws

import (
	"context"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/sts"
)

func CheckLogin(profile string) error {
	cfg, err := config.LoadDefaultConfig(
		context.Background(),
		config.WithSharedConfigProfile(profile),
	)
	if err != nil {
		return err
	}

	client := sts.NewFromConfig(cfg)

	_, err = client.GetCallerIdentity(context.Background(), &sts.GetCallerIdentityInput{})
	if err != nil {
		return fmt.Errorf(
			"AWS SSO session expired or not logged in.\n\n"+
				"Please run:\n\n  aws sso login --profile %s\n",
			profile,
		)
	}

	return nil
}
