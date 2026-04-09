package aws

import (
	"context"

	"cpctl/internal/providers"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/ssm"
)

type SSMProvider struct {
	Path string
}

func NormalizeProviderSecrets(items []providers.SecretItem) map[string]string {
	out := map[string]string{}
	for _, item := range items {
		out[item.Name] = string(item.Value)
	}
	return out
}

func (p *SSMProvider) List() ([]providers.SecretItem, error) {
	cfg, err := config.LoadDefaultConfig(context.TODO())
	if err != nil {
		return nil, err
	}

	client := ssm.NewFromConfig(cfg)

	var items []providers.SecretItem
	var nextToken *string

	for {
		out, err := client.GetParametersByPath(context.TODO(), &ssm.GetParametersByPathInput{
			Path:           &p.Path,
			WithDecryption: aws.Bool(true),
			Recursive:      aws.Bool(true),
			NextToken:      nextToken,
		})
		if err != nil {
			return nil, err
		}

		for _, param := range out.Parameters {
			items = append(items, providers.SecretItem{
				Name:  *param.Name,
				Value: []byte(*param.Value),
			})
		}

		if out.NextToken == nil {
			break
		}
		nextToken = out.NextToken
	}

	return items, nil
}
