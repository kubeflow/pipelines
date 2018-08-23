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

import com.google.gson.Gson
import java.io._
import ml.dmlc.xgboost4j.scala.spark.XGBoost
import org.apache.spark.ml.linalg.Vector
import org.apache.spark.sql.expressions.UserDefinedFunction
import org.apache.spark.sql.functions._
import org.apache.spark.sql.SparkSession
import org.apache.spark.SparkConf
import org.apache.spark.SparkContext
import scala.sys.process.Process
import scala.util.parsing.json.JSON


/** A distributed XGBoost predictor program running in spark cluster.
 *  Args:
 *     model-path: GCS path of the trained xgboost model.
 *     predict-data-path: GCS path of the prediction libsvm file pattern.
 *     num-workers: number of spark worker node used for training.
 *     analysis-path: GCS path of analysis results directory.
 *     target-name: column name of the prediction target.
 *     output-path: GCS path to store the prediction results.
 */


case class SchemaEntry(name: String, `type`: String)


object XGBoostPredictor {

  // TODO: create a common class for the util functions.
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

  def getVocab(vocabFile: String): Array[String] = {
    val sparkSession = SparkSession.builder().getOrCreate()
    val vocabContent = sparkSession.sparkContext.wholeTextFiles(vocabFile).map(
        tuple => tuple._2).collect()(0)
    val vocabFreq = vocabContent.split("\n")
    val vocab = for (e <- vocabFreq) yield e.split(",")(0)
    vocab
  }

  def labelIndexToStringUdf(vocab: Array[String]): UserDefinedFunction = {
    val lookup: (Double => String) = (label: Double) => (vocab(label.toInt))
    udf(lookup)
  }

  def probsToPredictionUdf(vocab: Array[String]): UserDefinedFunction = {
    val convert: (Double => String) = (prob: Double) => (if (prob >= 0.5) vocab(1) else vocab(0))
    udf(convert)
  }

  def writeSchemaFile(output: String, schema: Any): Unit = {
    val gson = new Gson
    val content = gson.toJson(schema)
    val pw = new PrintWriter(new File("schema.json" ))
    pw.write(content)
    pw.close()
    Process("gsutil cp schema.json " + output + "/schema.json").run
  }

  def main(args: Array[String]): Unit = {
    if (args.length != 5) {
      println(
        "usage: program model-path predict-data-path analysis-path " +
        "target-name, output-path")
      sys.exit(1)
    }

    val sparkSession = SparkSession.builder().getOrCreate()

    val modelPath = args(0)
    val inputPredictPath = args(1)
    val analysisPath = args(2)
    val targetName = args(3)
    val outputPath = args(4)

    // build dataset
    val feature_size = get_feature_size(analysisPath + "/stats.json", targetName)
    val predictDF = (sparkSession.sqlContext.read.format("libsvm")
                     .option("numFeatures", feature_size.toString).load(inputPredictPath))

    println("start prediction -------\n")
    implicit val sc = SparkContext.getOrCreate()
    val xgbModel = XGBoost.loadModelFromHadoopFile(modelPath)
    val predictResultsDF = xgbModel.transform(predictDF)

    val isClassification = isClassificationTask(analysisPath + "/schema.json", targetName)
    if (isClassification) {
      val targetVocab = getVocab(analysisPath + "/vocab_" + targetName + ".csv")
      val lookupUdf = labelIndexToStringUdf(targetVocab)
      var processedDF = (predictResultsDF.withColumn("target", lookupUdf(col("label")))
                                         .withColumn("predicted", lookupUdf(col("prediction"))))
      var schema = Array(SchemaEntry("target", "CATEGORY"),
                         SchemaEntry("predicted", "CATEGORY"))
      var columns = Array("target", "predicted")
      if (predictResultsDF.columns.contains("probabilities")) {
        // probabilities column includes array of probs for each class. Need to expand it.
        var classIndex = 0
        for (classname <- targetVocab) {
          // Need to make a val copy because the value of "classIndex" is evaluated at "run"
          // later when the resulting dataframe is used.
          val classIndexCopy = classIndex
          val extractValue = udf((arr: Vector) => arr(classIndexCopy))
          processedDF = processedDF.withColumn(classname, extractValue(col("probabilities")))
          schema :+= SchemaEntry(classname, "NUMBER")
          classIndex += 1
        }
        columns = columns ++ (for (e <- targetVocab) yield "`" + e + "`")
      }
      processedDF.select(columns.map(col): _*).write.option("header", "false").csv(outputPath)
      writeSchemaFile(outputPath, schema)
    } else {
      // regression
      val processedDF = (predictResultsDF.withColumnRenamed("prediction", "predicted")
                                         .withColumnRenamed("label", "target"))
      processedDF.select("target", "predicted").write.option("header", "false").csv(outputPath)
      val schema = Array(SchemaEntry("target", "NUMBER"),
                         SchemaEntry("predicted", "NUMBER"))
      writeSchemaFile(outputPath, schema)
    }
  }
}
