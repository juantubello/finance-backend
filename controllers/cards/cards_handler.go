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

type BalanceController struct {
	*cards.BaseController // Embed base to share base methods
}

type resumePaths struct {
	cardLogo string
	filePath string
	fileName string
}

func NewCardsController() *BalanceController {
	return &BalanceController{
		BaseController: &cards.BaseController{},
	}
}

func (ec *BalanceController) SyncResumes(c *gin.Context) {

	resumesPath := getResumesFilePath()

	c.JSON(http.StatusOK, gin.H{"test": resumesPath})
}

func getResumesFilePath() []resumePaths {

	var resumes []resumePaths
	//add len to directories
	directories := make([]string, 2)

	directories = append(directories, config.GetEnv("CARD_VISA_PATH"))
	directories = append(directories, config.GetEnv("CARD_MASTERCARD_PATH"))

	for _, dir := range directories {

		entries, err := os.ReadDir(dir)
		if err != nil {
			//  log.Fatal(err)
		}

		//var files []string
		for _, v := range entries {
			if filepath.Ext(v.Name()) != ".pdf" {
				continue
			}
			//completePath := fmt.Sprintf("%s/%s", dir, v.Name())
			completePath := dir + "/" + v.Name()
			fmt.Println(completePath)
			//		if v.IsDir() {
			//			continue
			//		}
			//		if filepath.Ext(v.Name()) != ".md" {
			//			continue
			//		}
			//		files = append(files, filepath.Join(dir, v.Name()))
		}

	}

	//return files

	return resumes
}
