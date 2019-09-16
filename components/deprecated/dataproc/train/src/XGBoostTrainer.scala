/*
 Copyright 2018 Google LLC

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


package ml.dmlc.xgboost4j.scala.example.spark

import ml.dmlc.xgboost4j.scala.Booster
import ml.dmlc.xgboost4j.scala.spark.XGBoost
import org.apache.spark.sql.SparkSession
import org.apache.spark.SparkConf
import org.apache.spark.SparkContext
import org.apache.spark.sql.functions.col
import scala.util.parsing.json.JSON


/** A distributed XGBoost trainer program running in spark cluster.
 *  Args:
 *     train-conf: GCS path of the training config json file for xgboost training.
 *     num-of-rounds: number of rounds to train.
 *     num-workers: number of spark worker node used for training.
 *     analysis-path: GCS path of analysis results directory.
 *     target-name: column name of the prediction target.
 *     training-path: GCS path of training libsvm file patterns.
 *     eval-path: GCS path of eval libsvm file patterns.
 *     output-path: GCS path to store the trained model.
 */


object XGBoostTrainer {

  def column_feature_size(stats: (String, Any), target: String): Double = {
    if (stats._1 == target) 0.0
    val statsMap = stats._2.asInstanceOf[Map[String, Any]]
    if (statsMap.keys.exists(_ == "vocab_size")) statsMap("vocab_size").asInstanceOf[Double]
    else if (statsMap.keys.exists(_ == "max")) 1.0
    else 0.0
  }

  def get_feature_size(statsPath: String, target: String): Int = {
    val sparkSession = SparkSession.builder().getOrCreate()
    val schema_string = sparkSession.sparkContext.wholeTextFiles(
        statsPath).map(tuple => tuple._2).collect()(0)
    val column_stats = JSON.parseFull(schema_string).get.asInstanceOf[Map[String, Any]](
        "column_stats").asInstanceOf[Map[String, Any]]
    var sum = 0.0
    for (stats <- column_stats) sum = sum + column_feature_size(stats, target)
    sum.toInt
  }

  def read_config(configFile: String): Map[String, Any] = {
    val sparkSession = SparkSession.builder().getOrCreate()
    val confString = sparkSession.sparkContext.wholeTextFiles(
        configFile).map(tuple => tuple._2).collect()(0)

    // Avoid parsing "500" to "500.0"
    val originNumberParser = JSON.perThreadNumberParser
    JSON.perThreadNumberParser = {
        in => try in.toInt catch { case _: NumberFormatException => in.toDouble}
    }
    try JSON.parseFull(confString).get.asInstanceOf[Map[String, Any]] finally {
      JSON.perThreadNumberParser = originNumberParser
    }
  }

  def isClassificationTask(schemaFile: String, targetName: String): Boolean = {
    val sparkSession = SparkSession.builder().getOrCreate()
    val schemaString = sparkSession.sparkContext.wholeTextFiles(
        schemaFile).map(tuple => tuple._2).collect()(0)
    val schema = JSON.parseFull(schemaString).get.asInstanceOf[List[Map[String, String]]]
    val targetList = schema.filter(x => x("name") == targetName)
    if (targetList.isEmpty) {
      throw new IllegalArgumentException("target cannot be found.")
    }
    val targetType = targetList(0)("type")
    if (targetType == "CATEGORY") true
    else if (targetType == "NUMBER") false
    else throw new IllegalArgumentException("invalid target type.")
  }

  def main(args: Array[String]): Unit = {
    if (args.length != 8) {
      println(
        "usage: program train-conf num-of-rounds num-workers analysis-path " +
        "target-name training-path eval-path output-path")
      sys.exit(1)
    }

    val sparkSession = SparkSession.builder().getOrCreate()

    val trainConf = args(0)
    val numRounds = args(1).toInt
    val numWorkers = args(2).toInt
    val analysisPath = args(3)
    val targetName = args(4)
    val inputTrainPath = args(5)
    val inputTestPath = args(6)
    val outputPath = args(7)

    // build dataset
    val feature_size = get_feature_size(analysisPath + "/stats.json", targetName)
    val trainDF = sparkSession.sqlContext.read.format("libsvm").option(
        "numFeatures", feature_size.toString).load(inputTrainPath)
    val testDF = sparkSession.sqlContext.read.format("libsvm").option(
        "numFeatures", feature_size.toString).load(inputTestPath)

    // start training
    val paramMap = read_config(trainConf)
    val xgboostModel = XGBoost.trainWithDataFrame(
        trainDF, paramMap, numRounds, nWorkers = numWorkers, useExternalMemory = true)

    println("training summary -------\n")
    println(xgboostModel.summary)

    // xgboost-spark appends the column containing prediction results
    val predictionDF = xgboostModel.transform(testDF)

    val classification = isClassificationTask(analysisPath + "/schema.json", targetName)
    implicit val sc = SparkContext.getOrCreate()
    if (classification) {
      val correctCounts = predictionDF.filter(
          col("prediction") === col("label")).groupBy(col("label")).count.collect
      val totalCounts = predictionDF.groupBy(col("label")).count.collect
      val accuracyAll = (predictionDF.filter(col("prediction") === col("label")).count /
                         predictionDF.count.toDouble)
      print("\naccuracy: " + accuracyAll + "\n")
    } else {
      predictionDF.createOrReplaceTempView("prediction")
      val rmseDF = sparkSession.sql(
          "SELECT SQRT(AVG((prediction - label) * (prediction - label))) FROM prediction")
      val rmse = rmseDF.collect()(0).getDouble(0)
      print("RMSE: " + rmse + "\n")
    }

    xgboostModel.saveModelAsHadoopFile(outputPath)
    print("Done")
  }
}
