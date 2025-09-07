package usecase

import (
	"context"

	cron_adapter "github.com/MMN3003/mega/src/order/adapter/cron"
	"github.com/MMN3003/mega/src/order/domain"
	"github.com/google/uuid"
	"github.com/robfig/cron/v3"
)

var (
	PendingOrdersCronID            = uuid.MustParse("62444ba0-b2dd-4b8f-afee-c04f7b2ab6e0")
	SuccessDebitCronID             = uuid.MustParse("62444ba0-b2dd-4b8f-afee-c04f7b2ab6e1")
	ReturnUserOrdersID             = uuid.MustParse("62444ba0-b2dd-4b8f-afee-c04f7b2ab6e2")
	MarketUserOrderSuccessOrdersID = uuid.MustParse("62444ba0-b2dd-4b8f-afee-c04f7b2ab6e3")
	MarketUserOrderFailedOrdersID  = uuid.MustParse("62444ba0-b2dd-4b8f-afee-c04f7b2ab6e4")
)

func NewCronService(c *cron.Cron, s domain.OrderUsecase, ca cron_adapter.CronAdapter) {
	c.AddFunc("1 * * * * *", func() {
		handlePendingOrders(context.Background(), s, ca)
	})
	c.AddFunc("1 * * * * *", func() {
		handleSuccessDebitOrders(context.Background(), s, ca)
	})
	c.AddFunc("1 * * * * *", func() {
		handleReturnUserOrders(context.Background(), s, ca)
	})
	c.AddFunc("1 * * * * *", func() {
		handleMarketUserOrderSuccessOrders(context.Background(), s, ca)
	})
	c.AddFunc("1 * * * * *", func() {
		handleFailedMarketUserOrderOrders(context.Background(), s, ca)
	})
}

func handlePendingOrders(ctx context.Context, o domain.OrderUsecase, ca cron_adapter.CronAdapter) {

	err := ca.CreateCron(ctx, PendingOrdersCronID)
	if err != nil {
		return
	}
	o.FetchPendingOrders(ctx)

	err = ca.DeleteCron(ctx, PendingOrdersCronID)
	if err != nil {
		return
	}
}

func handleSuccessDebitOrders(ctx context.Context, o domain.OrderUsecase, ca cron_adapter.CronAdapter) {
	err := ca.CreateCron(ctx, SuccessDebitCronID)
	if err != nil {
		return
	}
	o.FetchSuccessDebitOrders(ctx)

	err = ca.DeleteCron(ctx, SuccessDebitCronID)
	if err != nil {
		return
	}
}

func handleReturnUserOrders(ctx context.Context, o domain.OrderUsecase, ca cron_adapter.CronAdapter) {
	err := ca.CreateCron(ctx, ReturnUserOrdersID)
	if err != nil {
		return
	}
	o.FetchReturnUserOrders(ctx)

	err = ca.DeleteCron(ctx, ReturnUserOrdersID)
	if err != nil {
		return
	}
}

func handleMarketUserOrderSuccessOrders(ctx context.Context, o domain.OrderUsecase, ca cron_adapter.CronAdapter) {
	err := ca.CreateCron(ctx, MarketUserOrderSuccessOrdersID)
	if err != nil {
		return
	}
	o.FetchMarketUserOrderSuccessOrders(ctx)

	err = ca.DeleteCron(ctx, MarketUserOrderSuccessOrdersID)
	if err != nil {
		return
	}
}
func handleFailedMarketUserOrderOrders(ctx context.Context, o domain.OrderUsecase, ca cron_adapter.CronAdapter) {
	err := ca.CreateCron(ctx, MarketUserOrderFailedOrdersID)
	if err != nil {
		return
	}
	o.FetchFailedMarketUserOrderOrders(ctx)

	err = ca.DeleteCron(ctx, MarketUserOrderFailedOrdersID)
	if err != nil {
		return
	}
}
