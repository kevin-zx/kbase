package kfeishu

import (
	"context"
	"fmt"

	"github.com/kevin-zx/kbase/kfeishu/token"
	lark "github.com/larksuite/oapi-sdk-go/v3"
	larkauth "github.com/larksuite/oapi-sdk-go/v3/service/auth/v3"
	larkauthen "github.com/larksuite/oapi-sdk-go/v3/service/authen/v1"
)

type FeishuClient interface {
	GetUserAccessToken() (string, error)
	GetUserAccessTokenByAuthCode(code string) (string, error)
	RefreshUserAccessToken(rtoken token.UserAccessToken) (string, error)
	GetAppAccessToken() (*AppAceessTokenResp, error)
	GetClient() *lark.Client
}

type feishuClient struct {
	client    *lark.Client
	AppID     string
	AppSecret string
	ts        token.TokenService
}

func NewFeishuClient(ts token.TokenService, appId, appSecret string) feishuClient {
	client := lark.NewClient(appId, appSecret)
	fc := feishuClient{
		client:    client,
		AppID:     appId,
		AppSecret: appSecret,
		ts:        ts,
	}
	return fc
}

type AppAceessTokenResp struct {
	AppAccessToken string `json:"app_access"`
	Expire         int    `json:"expire"`
}

// get user access token
func (f *feishuClient) GetUserAccessToken() (string, error) {
	token, err := f.ts.GetUserAccessToken()
	if err != nil {
		return "", err
	}
	if token != nil && !token.IsExpired() {
		return token.AccessToken, nil
	}
	if token != nil && token.IsExpired() && !token.IsRefreshTokenExpired() {
		return f.RefreshUserAccessToken(*token)
	}

	authCode, err := f.ts.GetAuthCode()
	if err != nil {
		return "", err
	}
	if authCode == nil {
		return "", fmt.Errorf("auth code not found, please authorize first by url")
	}

	return f.GetUserAccessTokenByAuthCode(authCode.Code)
}

// get user access token by auth code
func (f *feishuClient) GetUserAccessTokenByAuthCode(code string) (string, error) {
	req := larkauthen.NewCreateOidcAccessTokenReqBuilder().
		Body(larkauthen.NewCreateOidcAccessTokenReqBodyBuilder().
			GrantType(`authorization_code`).
			Code(code).
			Build()).
		Build()
	resp, err := f.client.Authen.OidcAccessToken.Create(context.Background(), req)
	if err != nil {
		return "", err
	}
	if !resp.Success() {
		return "", fmt.Errorf("code: %d, msg: %s, requestId: %s", resp.Code, resp.Msg, resp.RequestId())
	}
	accessTokenDao := &token.UserAccessToken{
		AccessToken:           *resp.Data.AccessToken,
		ExpiresIn:             int64(*resp.Data.ExpiresIn),
		RefreshToken:          *resp.Data.RefreshToken,
		RefreshTokenExpiresIn: int64(*resp.Data.RefreshExpiresIn),
		Scope:                 *resp.Data.Scope,
	}
	err = f.ts.CreateUserAccessToken(accessTokenDao)
	if err != nil {
		return "", err
	}
	return *resp.Data.AccessToken, nil
}

// refresh user access token
func (f *feishuClient) RefreshUserAccessToken(rtoken token.UserAccessToken) (string, error) {
	req := larkauthen.NewCreateOidcRefreshAccessTokenReqBuilder().
		Body(larkauthen.NewCreateOidcRefreshAccessTokenReqBodyBuilder().
			GrantType(`refresh_token`).
			RefreshToken(rtoken.RefreshToken).
			Build()).
		Build()
	resp, err := f.client.Authen.OidcRefreshAccessToken.Create(context.Background(), req)
	if err != nil {
		return "", err
	}
	if !resp.Success() {
		return "", fmt.Errorf("code: %d, msg: %s, requestId: %s", resp.Code, resp.Msg, resp.RequestId())
	}
	accessTokenDao := &token.UserAccessToken{
		AccessToken:           *resp.Data.AccessToken,
		ExpiresIn:             int64(*resp.Data.ExpiresIn),
		RefreshToken:          *resp.Data.RefreshToken,
		RefreshTokenExpiresIn: int64(*resp.Data.RefreshExpiresIn),
		Scope:                 *resp.Data.Scope,
	}
	err = f.ts.CreateUserAccessToken(accessTokenDao)
	if err != nil {
		return "", err
	}
	return *resp.Data.AccessToken, nil

}

// get access token
func (f *feishuClient) GetAppAccessToken() (*AppAceessTokenResp, error) {
	req := larkauth.NewInternalAppAccessTokenReqBuilder().
		Body(larkauth.NewInternalAppAccessTokenReqBodyBuilder().
			AppId(f.AppID).
			AppSecret(f.AppSecret).
			Build()).
		Build()

	// 发起请求
	resp, err := f.client.Auth.AppAccessToken.Internal(context.Background(), req)
	if err != nil {
		return nil, err
	}

	if !resp.Success() {
		return nil, fmt.Errorf("code: %d, msg: %s, requestId: %s", resp.Code, resp.Msg, resp.RequestId())
	}
	return nil, nil

}
