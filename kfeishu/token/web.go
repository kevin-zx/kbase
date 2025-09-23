package token

import (
	"log"
	"net/http"
	"net/url"
	"strings"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type FeishuTokenWebService interface {
	Run() error
}

type feishuTokenService struct {
	db          *gorm.DB
	ts          TokenService
	feishuAppID string
	redirectURI string
}

var defaultScopes = []string{
	"sheets:spreadsheet",
	"bitable:app",
	"docx:document",
	"docx:document.block:convert",
}

func NewFeishuTokenWebService(db *gorm.DB, feishuAppID, redirectURI string) FeishuTokenWebService {
	ts := NewTokenService(db)
	return &feishuTokenService{
		db:          db,
		ts:          ts,
		feishuAppID: feishuAppID,
		redirectURI: redirectURI,
	}
}

func (s *feishuTokenService) Run() error {
	db := s.db
	err := db.AutoMigrate(&AuthCode{})
	if err != nil {
		log.Println(err)
		return err
	}
	s.db = db
	r := gin.Default()
	r.GET("/authorize", s.GetAuthCode)
	r.GET("/authorization", s.Authorization)
	// newest token
	r.GET("/newest_token", s.GetNewestUserAccessToken)
	r.Run(":4080")
	return nil

}

var authURL = "https://open.feishu.cn/open-apis/authen/v1/authorize"

// authorize
func (s *feishuTokenService) GetAuthCode(c *gin.Context) {
	// appID := config.Conf.Feishu.AppID
	// redirectURI := config.Conf.Feishu.RedirectURI
	scopes := strings.Join(defaultScopes, " ")
	queryValues := url.Values{
		"app_id":       {s.feishuAppID},
		"redirect_uri": {s.redirectURI},
		"state":        {"state"},
		"scope":        {scopes},
	}

	authURL := authURL + "?" + queryValues.Encode()

	c.Redirect(http.StatusFound, authURL)
}

// get newest user access token
func (s *feishuTokenService) GetNewestUserAccessToken(c *gin.Context) {
	token, err := s.ts.GetUserAccessToken()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"message": "get newest user access token failed",
		})
		log.Println(err)
		return
	}
	if token == nil {
		c.JSON(http.StatusNotFound, gin.H{
			"message": "no user access token found",
		})
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"access_token": token.AccessToken,
		"expires_in":   token.ExpiresIn,
		"create_time":  token.CreatedAt.Unix(),
	})
}

// authorization
// 接受code，存入数据库
func (s *feishuTokenService) Authorization(c *gin.Context) {
	code := c.Query("code")
	state := c.DefaultQuery("state", "default_state")
	if code == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"message": "code is required",
		})
		return
	}
	acode := AuthCode{
		Code:  code,
		State: state,
	}
	err := s.db.Create(&acode).Error
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"message": "create auth code failed",
		})
		log.Println(err)
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"message": "success",
	})
}
