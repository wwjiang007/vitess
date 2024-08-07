/*
Copyright 2023 The Vitess Authors.

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

package dml

import (
	"fmt"
	"math/rand/v2"
	"strconv"
	"strings"
	"testing"

	"vitess.io/vitess/go/test/endtoend/utils"
)

type testQuery struct {
	tableName string
	cols      []string
	intTyp    []bool
}

var deleteUser, deleteUserExtra = "delete from user", "delete from user_extra"

func generateInserts(userSize int, userExtraSize int) (string, string) {
	var userInserts []string
	var userExtraInserts []string

	// Generate user table inserts
	for i := 1; i <= userSize; i++ {
		id := i
		notShardingKey := i
		typeValue := i % 5            // Just an example for type values
		teamID := i%userExtraSize + 1 // To ensure team_id references user_extra id
		userInserts = append(userInserts, fmt.Sprintf("(%d, %d, %d, %d)", id, notShardingKey, typeValue, teamID))
	}

	// Generate user_extra table inserts
	for i := 1; i <= userExtraSize; i++ {
		id := i
		notShardingKey := i
		colValue := fmt.Sprintf("col_value_%d", i)
		userExtraInserts = append(userExtraInserts, fmt.Sprintf("(%d, %d, '%s')", id, notShardingKey, colValue))
	}

	userInsertStatement := fmt.Sprintf("INSERT INTO user (id, not_sharding_key, type, team_id) VALUES %s;", strings.Join(userInserts, ", "))
	userExtraInsertStatement := fmt.Sprintf("INSERT INTO user_extra (id, not_sharding_key, col) VALUES %s;", strings.Join(userExtraInserts, ", "))

	return userInsertStatement, userExtraInsertStatement
}

func (tq *testQuery) getInsertQuery(rows int) string {
	var allRows []string
	for i := 0; i < rows; i++ {
		var row []string
		for _, isInt := range tq.intTyp {
			if isInt {
				row = append(row, strconv.Itoa(i))
				continue
			}
			row = append(row, "'"+getRandomString(50)+"'")
		}
		allRows = append(allRows, "("+strings.Join(row, ",")+")")
	}
	return fmt.Sprintf("insert into %s(%s) values %s", tq.tableName, strings.Join(tq.cols, ","), strings.Join(allRows, ","))
}

func (tq *testQuery) getUpdateQuery(rows int) string {
	var allRows []string
	var row []string
	for i, isInt := range tq.intTyp {
		if isInt {
			row = append(row, strconv.Itoa(i))
			continue
		}
		row = append(row, tq.cols[i]+" = '"+getRandomString(50)+"'")
	}
	allRows = append(allRows, strings.Join(row, ","))

	var ids []string
	for i := 0; i <= rows; i++ {
		ids = append(ids, strconv.Itoa(i))
	}
	return fmt.Sprintf("update %s set %s where id in (%s)", tq.tableName, strings.Join(allRows, ","), strings.Join(ids, ","))
}

func (tq *testQuery) getDeleteQuery(rows int) string {
	var ids []string
	for i := 0; i <= rows; i++ {
		ids = append(ids, strconv.Itoa(i))
	}
	return fmt.Sprintf("delete from %s where id in (%s)", tq.tableName, strings.Join(ids, ","))
}

func getRandomString(size int) string {
	var str strings.Builder

	for i := 0; i < size; i++ {
		str.WriteByte(byte(rand.IntN(27) + 97))
	}
	return str.String()
}

func BenchmarkShardedTblNoLookup(b *testing.B) {
	conn, closer := start(b)
	defer closer()

	cols := []string{"id", "c1", "c2", "c3", "c4", "c5", "c6", "c7", "c8", "c9", "c10", "c11", "c12"}
	intType := make([]bool, len(cols))
	intType[0] = true
	tq := &testQuery{
		tableName: "tbl_no_lkp_vdx",
		cols:      cols,
		intTyp:    intType,
	}
	for _, rows := range []int{1, 10, 100, 500, 1000, 5000, 10000} {
		insStmt := tq.getInsertQuery(rows)
		b.Run(fmt.Sprintf("16-shards-%d-rows", rows), func(b *testing.B) {
			for i := 0; i < b.N; i++ {
				_ = utils.Exec(b, conn, insStmt)
			}
		})
	}
}

func BenchmarkShardedTblUpdateIn(b *testing.B) {
	conn, closer := start(b)
	defer closer()

	cols := []string{"c1", "c2", "c3", "c4", "c5", "c6", "c7", "c8", "c9", "c10", "c11", "c12"}
	intType := make([]bool, len(cols))
	tq := &testQuery{
		tableName: "tbl_no_lkp_vdx",
		cols:      cols,
		intTyp:    intType,
	}
	insStmt := tq.getInsertQuery(10000)
	_ = utils.Exec(b, conn, insStmt)
	for _, rows := range []int{1, 10, 100, 500, 1000, 5000, 10000} {
		updStmt := tq.getUpdateQuery(rows)
		b.Run(fmt.Sprintf("16-shards-%d-rows", rows), func(b *testing.B) {
			for i := 0; i < b.N; i++ {
				_ = utils.Exec(b, conn, updStmt)
			}
		})
	}
}

func BenchmarkShardedTblDeleteIn(b *testing.B) {
	conn, closer := start(b)
	defer closer()
	tq := &testQuery{
		tableName: "tbl_no_lkp_vdx",
	}
	for _, rows := range []int{1, 10, 100, 500, 1000, 5000, 10000} {
		insStmt := tq.getInsertQuery(rows)
		_ = utils.Exec(b, conn, insStmt)
		delStmt := tq.getDeleteQuery(rows)
		b.Run(fmt.Sprintf("16-shards-%d-rows", rows), func(b *testing.B) {
			for i := 0; i < b.N; i++ {
				_ = utils.Exec(b, conn, delStmt)
			}
		})
	}
}

func BenchmarkShardedAggrPushDown(b *testing.B) {
	conn, closer := start(b)
	defer closer()

	sizes := []int{100, 500, 1000}

	for _, user := range sizes {
		for _, userExtra := range sizes {
			insert1, insert2 := generateInserts(user, userExtra)
			_ = utils.Exec(b, conn, insert1)
			_ = utils.Exec(b, conn, insert2)
			b.Run(fmt.Sprintf("user-%d-user_extra-%d", user, userExtra), func(b *testing.B) {
				for i := 0; i < b.N; i++ {
					_ = utils.Exec(b, conn, "select sum(user.type) from user join user_extra on user.team_id = user_extra.id group by user_extra.id order by user_extra.id")
				}
			})
			_ = utils.Exec(b, conn, deleteUser)
			_ = utils.Exec(b, conn, deleteUserExtra)
		}
	}
}
