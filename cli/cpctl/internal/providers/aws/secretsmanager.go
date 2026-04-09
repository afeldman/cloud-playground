package aws

import (
	"context"
	"strings"

	"cpctl/internal/providers"

	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/secretsmanager"
)

type SecretsManagerProvider struct {
	Prefix string // optional
}

func (p *SecretsManagerProvider) List() ([]providers.SecretItem, error) {
	cfg, err := config.LoadDefaultConfig(context.TODO())
	if err != nil {
		return nil, err
	}

	client := secretsmanager.NewFromConfig(cfg)

	var items []providers.SecretItem
	var nextToken *string

	for {
		out, err := client.ListSecrets(context.TODO(), &secretsmanager.ListSecretsInput{
			NextToken: nextToken,
		})
		if err != nil {
			return nil, err
		}

		for _, s := range out.SecretList {
			if p.Prefix != "" && !strings.HasPrefix(*s.Name, p.Prefix) {
				continue
			}

			val, err := client.GetSecretValue(context.TODO(), &secretsmanager.GetSecretValueInput{
				SecretId: s.ARN,
			})
			if err != nil {
				return nil, err
			}

			items = append(items, providers.SecretItem{
				Name:  *s.Name,
				Value: []byte(*val.SecretString),
			})
		}

		if out.NextToken == nil {
			break
		}
		nextToken = out.NextToken
	}

	return items, nil
}
