/*
 * Copyright 2019 The Vitess Authors.

 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at

 *     http://www.apache.org/licenses/LICENSE-2.0

 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package io.vitess.jdbc;

import org.junit.Assert;
import org.junit.Test;

import java.sql.ParameterMetaData;
import java.sql.SQLException;
import java.sql.Types;

public class VitessParameterMetaDataTest {

  @Test
  public void testValidSimpleResponses() throws SQLException {
    VitessParameterMetaData metaData = new VitessParameterMetaData(5);
    Assert.assertEquals("parameterCount", 5, metaData.getParameterCount());
    Assert.assertEquals("parameterMode", ParameterMetaData.parameterModeIn,
        metaData.getParameterMode(2));
    Assert.assertEquals("parameterType", Types.VARCHAR, metaData.getParameterType(2));
    Assert.assertEquals("precision", 0, metaData.getPrecision(2));
    Assert.assertEquals("scale", 0, metaData.getScale(2));
    Assert
        .assertEquals("parameterClassName", "java.lang.String", metaData.getParameterClassName(2));
    Assert.assertEquals("parameterTypeName", "VARCHAR", metaData.getParameterTypeName(2));
    Assert.assertEquals("signed", false, metaData.isSigned(2));
  }

  @Test
  public void testOutOfBoundsValidation() {
    int parameterCount = 1;
    VitessParameterMetaData metaData = new VitessParameterMetaData(parameterCount);

    try {
      metaData.getParameterType(0);
      Assert.fail();
    } catch (SQLException e) {
      Assert.assertEquals("Parameter index of '0' is invalid.", e.getMessage());
    }

    int paramNumber = 2;
    try {
      metaData.getParameterType(paramNumber);
      Assert.fail();
    } catch (SQLException e) {
      Assert.assertEquals("Parameter index of '" + paramNumber
              + "' is greater than number of parameters, which is '" + parameterCount + "'.",
          e.getMessage());
    }
  }

  @Test(expected = SQLException.class)
  public void testNullableNotAvailable() throws SQLException {
    VitessParameterMetaData metaData = new VitessParameterMetaData(5);
    metaData.isNullable(3);
    Assert.fail();
  }
}
