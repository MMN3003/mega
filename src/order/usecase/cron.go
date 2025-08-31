package usecase

import (
	"context"

	cron_adapter "github.com/MMN3003/mega/src/order/adapter/cron"
	"github.com/MMN3003/mega/src/order/domain"
	"github.com/google/uuid"
	"github.com/robfig/cron/v3"
)

var (
	PendingOrdersCronID    = uuid.MustParse("62444ba0-b2dd-4b8f-afee-c04f7b2ab6e0")
	SuccessDebitCronID     = uuid.MustParse("62444ba0-b2dd-4b8f-afee-c04f7b2ab6e1")
	FailedTreasuryCreditID = uuid.MustParse("62444ba0-b2dd-4b8f-afee-c04f7b2ab6e2")
)

func NewCronService(c *cron.Cron, s domain.OrderUsecase, ca cron_adapter.CronAdapter) {
	c.AddFunc("1 * * * * *", func() {
		handlePendingOrders(context.Background(), s, ca)
	})
	c.AddFunc("1 * * * * *", func() {
		handleSuccessDebitOrders(context.Background(), s, ca)
	})
	c.AddFunc("1 * * * * *", func() {
		handleFailedTreasuryCreditOrders(context.Background(), s, ca)
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

func handleFailedTreasuryCreditOrders(ctx context.Context, o domain.OrderUsecase, ca cron_adapter.CronAdapter) {
	err := ca.CreateCron(ctx, FailedTreasuryCreditID)
	if err != nil {
		return
	}
	o.FetchFailedTreasuryCreditOrders(ctx)

	err = ca.DeleteCron(ctx, FailedTreasuryCreditID)
	if err != nil {
		return
	}
}
