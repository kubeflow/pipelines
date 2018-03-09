/*
 Copyright (c) 2014 by Contributors

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

import ml.dmlc.xgboost4j.scala.spark.XGBoost
import org.apache.spark.sql.SparkSession
import org.apache.spark.SparkConf
import org.apache.spark.SparkContext
import scala.util.parsing.json.JSON


object XGBoostPredictor {

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

  def main(args: Array[String]): Unit = {
    if (args.length != 6) {
      println(
        "usage: program model-path predict-data-path num-workers analysis-path " +
        "target-name, output-path")
      sys.exit(1)
    }

    val sparkSession = SparkSession.builder().getOrCreate()

    val modelPath = args(0)
    val inputPredictPath = args(1)
    val numWorkers = args(2).toInt
    val analysisPath = args(3)
    val targetName = args(4)
    val outputPath = args(5)

    // build dataset
    val feature_size = get_feature_size(analysisPath + "/stats.json", targetName)
    val predictDF = sparkSession.sqlContext.read.format("libsvm").option(
        "numFeatures", feature_size.toString).load(inputPredictPath)

    println("start prediction -------\n")
    implicit val sc = SparkContext.getOrCreate()
    val xgbModel = XGBoost.loadModelFromHadoopFile(modelPath)
    val predictResultsDF = xgbModel.transform(predictDF)

    println("saving results -------\n")
    predictResultsDF.select("label", "prediction").write.csv(outputPath)
    print("Done")
  }
}
