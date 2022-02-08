package monitor

import (
	"context"
	"errors"
	"fmt"
	"net/url"
	"time"

	"github.com/razorpay/trino-gateway/internal/boot"
	"github.com/razorpay/trino-gateway/internal/provider"
	"github.com/razorpay/trino-gateway/internal/utils"
	gatewayv1 "github.com/razorpay/trino-gateway/rpc/gateway"
)

type Core struct {
	gatewayBackendClient gatewayv1.BackendApi
}

type ICore interface {
	EvaluateBackendNewState(ctx *context.Context) (*BackendsNewState, error)
	MarkHealthyBackend(ctx *context.Context, b *gatewayv1.Backend) error
	MarkUnhealthyBackend(ctx *context.Context, b *gatewayv1.Backend) error
}

func NewCore(b gatewayv1.BackendApi) *Core {
	return &Core{gatewayBackendClient: b}
}

type BackendsNewState struct {
	Healthy   []*gatewayv1.Backend
	Unhealthy []*gatewayv1.Backend
}

func (c *Core) EvaluateBackendNewState(ctx *context.Context) (*BackendsNewState, error) {
	provider.Logger(*ctx).Debug("Evaluating new state for backends based on uptime schedules")
	provider.Logger(*ctx).Debug("Fetching all backends")
	backends, err := c.getAllBackends(ctx)
	if err != nil {
		return nil, err
	}
	var markHealthy []*gatewayv1.Backend
	var markUnhealthy []*gatewayv1.Backend
	for _, b := range backends {
		if b == nil {
			return nil, errors.New("nil pointer reference in twirp response")
		}
		isEligible, err := c.isCurrentTimeInCron(ctx, b.UptimeSchedule)
		if err != nil {
			provider.Logger(*ctx).WithError(err).Errorw(
				"Unable to parse cron expression in uptime schedule, marking as unhealthy",
				map[string]interface{}{"backend": b},
			)
		} else if isEligible {
			provider.Logger(*ctx).Debugw(
				"Evaluating health of backend based on availability and cluster load",
				map[string]interface{}{"backend": b})

			isHealthy, err := c.isBackendHealthy(ctx, b)
			if err != nil {
				provider.Logger(*ctx).WithError(err).Errorw(
					"Failure checking backend health",
					map[string]interface{}{"backend": b})
			} else if isHealthy {
				markHealthy = append(markHealthy, b)
				continue
			}
		}

		markUnhealthy = append(markUnhealthy, b)
	}

	return &BackendsNewState{
		Healthy:   markHealthy,
		Unhealthy: markUnhealthy,
	}, nil
}

// checks whether current time is int the specified cron schedule pattern
// it will also evaluate to false if the pattern is invalid
func (c *Core) isCurrentTimeInCron(ctx *context.Context, sched string) (bool, error) {
	return utils.IsTimeInCron(ctx, time.Now(), sched)
}

func (c *Core) getAllBackends(ctx *context.Context) ([]*gatewayv1.Backend, error) {
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

func (c *Core) MarkHealthyBackend(ctx *context.Context, b *gatewayv1.Backend) error {
	_, err := c.gatewayBackendClient.
		MarkHealthyBackend(*ctx, &gatewayv1.BackendMarkHealthyRequest{
			Id: b.GetId(),
		})
	return err
}

func (c *Core) MarkUnhealthyBackend(ctx *context.Context, b *gatewayv1.Backend) error {
	_, err := c.gatewayBackendClient.
		MarkUnhealthyBackend(*ctx, &gatewayv1.BackendMarkUnhealthyRequest{
			Id: b.GetId(),
		})
	return err
}

func (c *Core) isBackendHealthy(ctx *context.Context, b *gatewayv1.Backend) (bool, error) {
	isUp, err := c.isBackendUp(ctx, b)
	if err != nil {
		provider.Logger(*ctx).WithError(err).Errorw(
			"Failure fetching cluster ready state, assuming unhealthy",
			map[string]interface{}{"backend": b})
		return false, err
	} else if isUp {
		provider.Logger(*ctx).Debugw(
			"Cluster is up",
			map[string]interface{}{"backend": b})
		load, err := c.getBackendLoad(ctx, b)
		if err != nil {
			provider.Logger(*ctx).WithError(err).Errorw(
				"Failure evaluating current backend load, assuming unhealthy",
				map[string]interface{}{"backend": b})
			return false, err
		}

		if err = c.updateBackendClusterLoad(ctx, b.GetId(), load); err != nil {
			provider.Logger(*ctx).WithError(err).Errorw(
				"Error updating cluster load stats for backend",
				map[string]interface{}{"backend_id": b.GetId(), "load": load})
		}

		threshold := b.ThresholdClusterLoad

		if threshold == 0 || load <= threshold {
			provider.Logger(*ctx).Debugw(
				"Cluster load below threshold, assuming healthy",
				map[string]interface{}{"backend": b})
			return true, nil
		} else {
			provider.Logger(*ctx).Infow(
				"Cluster load above threshold",
				map[string]interface{}{"backend": b, "load": load, "threshold": threshold})
		}
	}
	provider.Logger(*ctx).Infow(
		"Cluster unhealthy, marked as inactive",
		map[string]interface{}{"backend": b})

	return false, nil
}

func (c *Core) isBackendUp(ctx *context.Context, b *gatewayv1.Backend) (bool, error) {
	provider.Logger(*ctx).Debugw(
		"Checking if backend is healthy",
		map[string]interface{}{"backend": b})
	trinoClient := &TrinoClient{
		user: boot.Config.Monitor.Trino.User,
		url:  url.URL{Scheme: b.GetScheme().Enum().String(), Host: b.GetHostname()},
	}
	isUp, err := trinoClient.IsClusterUp(ctx)
	if err != nil {
		provider.Logger(*ctx).WithError(err).Errorw(
			"error checking whether trino cluster is up",
			map[string]interface{}{"backend_id": b.GetId()})
		return false, err
	}
	if !isUp {
		return false, nil
	}
	h, err := trinoClient.IsClusterHealthy(ctx)
	if err != nil {
		provider.Logger(*ctx).WithError(err).Errorw(
			"error checking trino cluster health",
			map[string]interface{}{"backend_id": b.GetId()})
		return false, err
	}
	return h, nil
}

func (c *Core) updateBackendClusterLoad(ctx *context.Context, b_id string, load int32) error {
	defer func() {
		metrics.backendLoad.WithLabelValues(b_id).
			Set(float64(load))
	}()
	_, err := c.gatewayBackendClient.
		UpdateClusterLoadBackend(*ctx, &gatewayv1.BackendUpdateClusterLoadRequest{
			Id:          b_id,
			ClusterLoad: load,
		})
	return err
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

func (c *Core) getBackendLoad(ctx *context.Context, b *gatewayv1.Backend) (int32, error) {
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
		"Fetching Load info from trino cluster",
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
	res.ActiveNodes = 0    // TODO - via system.runtime.nodes

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
