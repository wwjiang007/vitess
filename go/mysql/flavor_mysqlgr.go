/*
Copyright 2021 The Vitess Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package mysql

import (
	"errors"
	"fmt"
	"math"

	"vitess.io/vitess/go/mysql/capabilities"
	"vitess.io/vitess/go/mysql/replication"
	"vitess.io/vitess/go/vt/proto/vtrpc"

	"vitess.io/vitess/go/vt/vterrors"

	"vitess.io/vitess/go/sqltypes"
)

// GRFlavorID is the string identifier for the MysqlGR flavor.
const GRFlavorID = "MysqlGR"

// ErrNoGroupStatus means no status for group replication.
var ErrNoGroupStatus = errors.New("no group status")

// mysqlGRFlavor implements the Flavor interface for Mysql.
type mysqlGRFlavor struct {
	mysqlFlavor
}

// newMysqlGRFlavor creates a new mysqlGR flavor.
func newMysqlGRFlavor(serverVersion string) flavor {
	return &mysqlGRFlavor{mysqlFlavor{serverVersion: serverVersion}}
}

// startReplicationCommand returns the command to start the replication.
// we return empty here since `START GROUP_REPLICATION` should be called by
// the external orchestrator
func (mysqlGRFlavor) startReplicationCommand() string {
	return ""
}

// restartReplicationCommands is disabled in mysqlGRFlavor
func (mysqlGRFlavor) restartReplicationCommands() []string {
	return []string{}
}

// startReplicationUntilAfter is disabled in mysqlGRFlavor
func (mysqlGRFlavor) startReplicationUntilAfter(pos replication.Position) string {
	return ""
}

// startSQLThreadUntilAfter is disabled in mysqlGRFlavor
func (mysqlGRFlavor) startSQLThreadUntilAfter(pos replication.Position) string {
	return ""
}

// stopReplicationCommand returns the command to stop the replication.
// we return empty here since `STOP GROUP_REPLICATION` should be called by
// the external orchestrator
func (mysqlGRFlavor) stopReplicationCommand() string {
	return ""
}

func (mysqlGRFlavor) resetReplicationCommand() string {
	return ""
}

// stopIOThreadCommand is disabled in mysqlGRFlavor
func (mysqlGRFlavor) stopIOThreadCommand() string {
	return ""
}

// stopSQLThreadCommand is disabled in mysqlGRFlavor
func (mysqlGRFlavor) stopSQLThreadCommand() string {
	return ""
}

// startSQLThreadCommand is disabled in mysqlGRFlavor
func (mysqlGRFlavor) startSQLThreadCommand() string {
	return ""
}

// resetReplicationCommands is disabled in mysqlGRFlavor
func (mysqlGRFlavor) resetReplicationCommands(c *Conn) []string {
	return []string{}
}

// resetReplicationParametersCommands is part of the Flavor interface.
func (mysqlGRFlavor) resetReplicationParametersCommands(c *Conn) []string {
	return []string{}
}

// setReplicationPositionCommands is disabled in mysqlGRFlavor
func (mysqlGRFlavor) setReplicationPositionCommands(pos replication.Position) []string {
	return []string{}
}

// status returns the result of the appropriate status command,
// with parsed replication position.
//
// Note: primary will skip this function, only replica will call it.
// TODO: Right now the GR's lag is defined as the lag between a node processing a txn
// and the time the txn was committed. We should consider reporting lag between current queueing txn timestamp
// from replication_connection_status and the current processing txn's commit timestamp
func (mysqlGRFlavor) status(c *Conn) (replication.ReplicationStatus, error) {
	res := replication.ReplicationStatus{}
	// Get primary node information
	query := `SELECT
		MEMBER_HOST,
		MEMBER_PORT
	FROM
		performance_schema.replication_group_members
	WHERE
		MEMBER_ROLE='PRIMARY' AND MEMBER_STATE='ONLINE'`
	err := fetchStatusForGroupReplication(c, query, func(values []sqltypes.Value) error {
		parsePrimaryGroupMember(&res, values)
		return nil
	})
	if err != nil {
		return replication.ReplicationStatus{}, err
	}

	query = `SELECT
		MEMBER_STATE
	FROM
		performance_schema.replication_group_members
	WHERE
		MEMBER_HOST=convert(@@hostname using ascii) AND MEMBER_PORT=@@port`
	var chanel string
	err = fetchStatusForGroupReplication(c, query, func(values []sqltypes.Value) error {
		state := values[0].ToString()
		switch state {
		case "ONLINE":
			chanel = "group_replication_applier"
		case "RECOVERING":
			chanel = "group_replication_recovery"
		default: // OFFLINE, ERROR, UNREACHABLE
			// If the member is not in healthy state, use max int as lag
			res.ReplicationLagSeconds = math.MaxUint32
		}
		return nil
	})
	if err != nil {
		return replication.ReplicationStatus{}, err
	}
	// if chanel is not set, it means the state is not ONLINE or RECOVERING
	// return partial result early
	if chanel == "" {
		return res, nil
	}

	// Populate IOState from replication_connection_status
	query = fmt.Sprintf(`SELECT SERVICE_STATE
		FROM performance_schema.replication_connection_status
		WHERE CHANNEL_NAME='%s'`, chanel)
	var connectionState replication.ReplicationState
	err = fetchStatusForGroupReplication(c, query, func(values []sqltypes.Value) error {
		connectionState = replication.ReplicationStatusToState(values[0].ToString())
		return nil
	})
	if err != nil {
		return replication.ReplicationStatus{}, err
	}
	res.IOState = connectionState
	// Populate SQLState from replication_connection_status
	var applierState replication.ReplicationState
	query = fmt.Sprintf(`SELECT SERVICE_STATE
		FROM performance_schema.replication_applier_status_by_coordinator
		WHERE CHANNEL_NAME='%s'`, chanel)
	err = fetchStatusForGroupReplication(c, query, func(values []sqltypes.Value) error {
		applierState = replication.ReplicationStatusToState(values[0].ToString())
		return nil
	})
	if err != nil {
		return replication.ReplicationStatus{}, err
	}
	res.SQLState = applierState

	// Collect lag information
	// we use the difference between the last processed transaction's commit time
	// and the end buffer time as the proxy to the lag
	query = fmt.Sprintf(`SELECT
		TIMESTAMPDIFF(SECOND, LAST_PROCESSED_TRANSACTION_ORIGINAL_COMMIT_TIMESTAMP, LAST_PROCESSED_TRANSACTION_END_BUFFER_TIMESTAMP)
	FROM
		performance_schema.replication_applier_status_by_coordinator
	WHERE
		CHANNEL_NAME='%s'`, chanel)
	err = fetchStatusForGroupReplication(c, query, func(values []sqltypes.Value) error {
		parseReplicationApplierLag(&res, values)
		return nil
	})
	if err != nil {
		return replication.ReplicationStatus{}, err
	}
	return res, nil
}

func parsePrimaryGroupMember(res *replication.ReplicationStatus, row []sqltypes.Value) {
	res.SourceHost = row[0].ToString()   /* MEMBER_HOST */
	res.SourcePort, _ = row[1].ToInt32() /* MEMBER_PORT */
}

func parseReplicationApplierLag(res *replication.ReplicationStatus, row []sqltypes.Value) {
	lagSec, err := row[0].ToUint32()
	// if the error is not nil, ReplicationLagSeconds will remain to be MaxUint32
	if err == nil {
		// Only set where there is no error
		// The value can be NULL when there is no replication applied yet
		res.ReplicationLagSeconds = lagSec
	}
}

func fetchStatusForGroupReplication(c *Conn, query string, onResult func([]sqltypes.Value) error) error {
	qr, err := c.ExecuteFetch(query, 100, true /* wantfields */)
	if err != nil {
		return err
	}
	// if group replication related query returns 0 rows, it means the group replication is not set up
	if len(qr.Rows) == 0 {
		return ErrNoGroupStatus
	}
	if len(qr.Rows) > 1 {
		return vterrors.Errorf(vtrpc.Code_INTERNAL, "unexpected results for %v: %v", query, qr.Rows)
	}
	return onResult(qr.Rows[0])
}

// primaryStatus returns the result of 'SHOW BINARY LOG STATUS',
// with parsed executed position.
func (mysqlGRFlavor) primaryStatus(c *Conn) (replication.PrimaryStatus, error) {
	return mysqlFlavor{}.primaryStatus(c)
}

// replicationNetTimeout is part of the Flavor interface.
func (mysqlGRFlavor) replicationNetTimeout(c *Conn) (int32, error) {
	return mysqlFlavor8{}.replicationNetTimeout(c)
}

func (mysqlGRFlavor) baseShowTables() string {
	return mysqlFlavor{}.baseShowTables()
}

func (mysqlGRFlavor) baseShowTablesWithSizes() string {
	return "" // Won't be used, as InnoDBTableSizes is defined, and schema.Engine will use that, instead.
}

// baseShowInnodbTableSizes is part of the Flavor interface.
func (mysqlGRFlavor) baseShowInnodbTableSizes() string {
	return InnoDBTableSizes
}

// supportsCapability is part of the Flavor interface.
func (f mysqlGRFlavor) supportsCapability(capability capabilities.FlavorCapability) (bool, error) {
	return capabilities.MySQLVersionHasCapability(f.serverVersion, capability)
}

func init() {
	flavorFuncs[GRFlavorID] = newMysqlGRFlavor
}
