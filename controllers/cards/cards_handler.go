package cards

import (
	"finance-backend/config"
	cards "finance-backend/controllers/base"
	"fmt"
	"net/http"
	"os"
	"path/filepath"

	"github.com/gin-gonic/gin"
)

type CardsController struct {
	*cards.BaseController // Embed base to share base methods
}

type resumePaths struct {
	CardLogo string `json:"cardLogo"`
	FilePath string `json:"filePath"`
	FileName string `json:"fileName"`
}

func NewCardsController() *CardsController {
	return &CardsController{
		BaseController: &cards.BaseController{},
	}
}

func (ec *CardsController) SyncResumes(c *gin.Context) {

	resumesPath, err := getResumesFilePath()

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"test": resumesPath})
}

func getResumesFilePath() ([]resumePaths, error) {

	type directoriesPath struct {
		path     string
		cardLogo string
	}

	var resumesPath []resumePaths
	directories := make([]directoriesPath, 2)
	directoryInfo := directoriesPath{
		path:     config.GetEnv("CARD_VISA_PATH"),
		cardLogo: "visa",
	}
	directories[0] = directoryInfo

	directoryInfo = directoriesPath{
		path:     config.GetEnv("CARD_MASTERCARD_PATH"),
		cardLogo: "mastercard",
	}

	directories[1] = directoryInfo

	for _, dir := range directories {
		entries, err := os.ReadDir(dir.path)
		if err != nil {
			return nil, fmt.Errorf("error reading directory %s: %w", dir, err)
		}
		for _, v := range entries {
			if filepath.Ext(v.Name()) != ".pdf" {
				continue
			}
			completePath := dir.path + "/" + v.Name()
			fmt.Println(completePath)
			resumesPath = append(resumesPath, resumePaths{
				CardLogo: dir.cardLogo,
				FilePath: completePath,
				FileName: v.Name()[:len(v.Name())-len(filepath.Ext(v.Name()))],
			})

		}
	}

	return resumesPath, nil
}
