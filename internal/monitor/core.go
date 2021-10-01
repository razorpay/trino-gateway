package monitor

import (
	"context"
	"errors"
	"fmt"
	"net/url"
	"time"

	"github.com/razorpay/trino-gateway/internal/boot"
	"github.com/razorpay/trino-gateway/internal/provider"
	gatewayv1 "github.com/razorpay/trino-gateway/rpc/gateway"
	"github.com/robfig/cron/v3"
)

type Core struct {
	gatewayBackendClient gatewayv1.BackendApi
}

type ICore interface {
	EvaluateUptimeSchedules(ctx *context.Context) error
	UpdateBackend(ctx *context.Context, b *gatewayv1.Backend) error
	GetAllBackends(ctx *context.Context) ([]*gatewayv1.Backend, error)
	GetActiveBackends(ctx *context.Context) ([]*gatewayv1.Backend, error)
	GetBackendLoad(ctx *context.Context, b *gatewayv1.Backend) (int32, error)
	IsBackendHealthy(ctx *context.Context, b *gatewayv1.Backend) (bool, error)
	computeClusterLoad(ctx *context.Context, stats *clusterLoadStats) int32
}

func NewCore(b gatewayv1.BackendApi) *Core {
	return &Core{gatewayBackendClient: b}
}

func (c *Core) EvaluateUptimeSchedules(ctx *context.Context) error {
	provider.Logger(*ctx).Debug("Fetching all backends")
	backends, err := c.GetAllBackends(ctx)
	if err != nil {
		return err
	}
	for _, b := range backends {
		if b == nil {
			return errors.New("nil pointer reference in twirp response")
		}
		b.IsEnabled, err = c.isCurrentTimeInCron(ctx, b.UptimeSchedule)
		if err != nil {
			provider.Logger(*ctx).WithError(err).Errorw(
				"Unable to parse cron expression in uptime schedule, skipping",
				map[string]interface{}{"backend": b},
			)
			continue
		}
		c.UpdateBackend(ctx, b)
	}
	return nil
}

func (c *Core) isCurrentTimeInCron(ctx *context.Context, sched string) (bool, error) {
	s, err := cron.ParseStandard(sched)
	if err != nil {
		return false, err
	}
	curr := time.Now()
	nextRun := s.Next(curr)

	provider.Logger(*ctx).Debugw(
		"Evaluated next valid ts from cron expression",
		map[string]interface{}{
			"current": curr,
			"nextRun": nextRun,
		},
	)

	return nextRun.Sub(curr).Minutes() <= 1, nil
}

func (c *Core) UpdateBackend(ctx *context.Context, b *gatewayv1.Backend) error {
	_, err := c.gatewayBackendClient.CreateOrUpdateBackend(*ctx, b)
	return err
}

func (c *Core) GetAllBackends(ctx *context.Context) ([]*gatewayv1.Backend, error) {
	provider.Logger(*ctx).Debug("fetching all backends")
	resp, err := c.gatewayBackendClient.ListAllBackends(*ctx, &gatewayv1.Empty{})
	if err != nil {
		return nil, err
	}
	for _, b := range resp.GetItems() {
		if b == nil {
			// TODO: will this branch ever execute ??
			provider.Logger(*ctx).Error(
				"found nil pointer reference while deserializing protoResp to structs")
			return nil, errors.New("nil pointer reference in twirp response")
		}
	}
	return resp.GetItems(), nil
}

func (c *Core) GetActiveBackends(ctx *context.Context) ([]*gatewayv1.Backend, error) {
	backends, err := c.GetAllBackends(ctx)
	if err != nil {
		return nil, err
	}
	provider.Logger(*ctx).Debug("filter active backends")
	var res []*gatewayv1.Backend
	for _, b := range backends {
		if b.GetIsEnabled() {
			res = append(res, b)
		}
	}

	return res, nil
}

func (c *Core) IsBackendHealthy(ctx *context.Context, b *gatewayv1.Backend) (bool, error) {
	provider.Logger(*ctx).Debugw(
		"Checking if backend is healthy",
		map[string]interface{}{"backend": b})
	trinoClient := &TrinoClient{
		user: boot.Config.Monitor.Trino.User,
		url:  url.URL{Scheme: b.GetScheme().Enum().String(), Host: b.GetHostname()},
	}
	h, err := trinoClient.IsClusterHealthy(ctx)
	if err != nil {
		provider.Logger(*ctx).WithError(err).Errorw(
			"error checking trinocluster health",
			map[string]interface{}{"backend_id": b.GetId()})
		return false, err
	}
	return h, nil
}

type clusterLoadStats struct {
	Queued              int32
	WaitingForResources int32
	Dispatching         int32
	Planning            int32
	Starting            int32
	Running             int32
	Finishing           int32
	AvgQueueTimeMs      int64
	AvgCpuLoad          int32
	ActiveNodes         int32
}

func (c *Core) GetBackendLoad(ctx *context.Context, b *gatewayv1.Backend) (int32, error) {
	trinoClient := &TrinoClient{
		user: boot.Config.Monitor.Trino.User,
		url:  url.URL{Scheme: b.GetScheme().Enum().String(), Host: b.GetHostname()},
		pass: boot.Config.Monitor.Trino.Password,
	}

	type StateStat struct {
		state string
		count int32
	}

	q := fmt.Sprint(
		"SELECT state, count(*)",
		" FROM system.runtime.queries",
		fmt.Sprintf(
			" WHERE user != '%s' AND state NOT IN ('FINISHED', 'FAILED')",
			boot.Config.Monitor.Trino.User),
		" GROUP BY state",
	)
	provider.Logger(*ctx).Debugw(
		"Fetching Query info from trino cluster",
		map[string]interface{}{"backend_id": b.GetId(), "query": q})
	rows, err := trinoClient.RunQuery(ctx, q)
	if err != nil {
		provider.Logger(*ctx).WithError(err).Errorw(
			"error executing trino query",
			map[string]interface{}{"query": q, "backend_id": b.GetId()})
		return 0, err
	}
	defer rows.Close()

	var stateStats []StateStat
	for rows.Next() {
		var res StateStat
		if err := rows.Scan(&res.state, &res.count); err != nil {
			provider.Logger(*ctx).WithError(err).Errorw(
				"error parsing trino query results",
				map[string]interface{}{"query": q, "backend_id": b.GetId()})
			return 0, err
		}
		stateStats = append(stateStats, res)
	}
	if err = rows.Err(); err != nil {
		provider.Logger(*ctx).WithError(err).Errorw(
			"error parsing trino query results",
			map[string]interface{}{"query": q, "backend_id": b.GetId()})
		return 0, err
	}

	res := &clusterLoadStats{}

	// https://github.com/trinodb/trino/blob/a20f0c4fec44ad6662d3d07a2617cc3ef2a50f52/core/trino-main/src/main/java/io/trino/connector/system/QuerySystemTable.java#L121
	// https://github.com/trinodb/trino/blob/a20f0c4fec44ad6662d3d07a2617cc3ef2a50f52/core/trino-main/src/main/java/io/trino/execution/SqlQueryExecution.java#L652

	// All query states
	// https://github.com/trinodb/trino/blob/fe608f2723842037ff620d612a706900e79c52c8/core/trino-main/src/main/java/io/trino/execution/QueryState.java
	for _, s := range stateStats {
		switch s.state {
		case "QUEUED":
			res.Queued = s.count
		case "WAITING_FOR_RESOURCES":
			res.WaitingForResources = s.count
		case "DISPATCHING":
			res.Dispatching = s.count
		case "PLANNING":
			res.Planning = s.count
		case "STARTING":
			res.Starting = s.count
		case "RUNNING":
			res.Running = s.count
		case "FINISHING":
			res.Finishing = s.count
		default:
		}
	}

	res.AvgQueueTimeMs = 0 // TODO - via system.runtime.queries
	res.AvgCpuLoad = 0     // TODO - ideally via Prom/VictoriaDb Trino connector
	res.ActiveNodes = 0    //TODO - via system.runtime.nodes

	load := c.computeClusterLoad(ctx, res)
	return load, nil
}

func (c *Core) computeClusterLoad(ctx *context.Context, stats *clusterLoadStats) int32 {

	running := stats.Running + stats.Planning + stats.Finishing + stats.Dispatching
	queued := stats.Queued + stats.Starting

	// TODO - factor in more metrics
	load := ((running * 2) + (queued*1)/3)

	return load
}
