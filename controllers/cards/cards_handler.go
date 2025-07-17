package cards

import (
	"finance-backend/models"
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"

	cards "finance-backend/controllers/base"
)

type BalanceController struct {
	*cards.BaseController // Embed base to share base methods
}

func NewCardsController() *BalanceController {
	return &BalanceController{
		BaseController: &cards.BaseController{},
	}
}

func (ec *BalanceController) SyncResumes(c *gin.Context) {

	var cards models.Expenses
	fmt.Println(cards)

	c.JSON(http.StatusOK, gin.H{"test": 10})
}
