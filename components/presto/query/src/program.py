# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
# http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.

import argparse
from pyhive import presto


def get_conn(host=None, catalog=None, schema=None, user=None, pwd=None):
  conn = presto.connect(
      host=host,
      port=443,
      protocol="https",
      catalog=catalog,
      schema=schema,
      username=user,
      password=pwd,
  )

  return conn


def query(conn, query):
  cursor = conn.cursor()
  cursor.execute(query)
  cursor.fetchall()


def main():
  parser = argparse.ArgumentParser()
  parser.add_argument("--host", type=str, help="Presto Host.")
  parser.add_argument(
      "--catalog", type=str, required=True, help="The name of the catalog."
  )
  parser.add_argument(
      "--schema", type=str, required=True, help="The name of the schema."
  )
  parser.add_argument(
      "--query",
      type=str,
      required=True,
      help="The SQL query statements to be executed in Presto.",
  )
  parser.add_argument(
      "--user", type=str, required=True, help="The user of the Presto."
  )
  parser.add_argument(
      "--pwd", type=str, required=True, help="The password of the Presto."
  )
  parser.add_argument(
      "--output",
      type=str,
      required=True,
      help="The path or name of the emitted output.",
  )

  args = parser.parse_args()

  conn = get_conn(args.host, args.catalog, args.schema, args.user, args.pwd)
  query(conn, args.query)

  with open("/output.txt", "w+") as w:
    w.write(args.output)


if __name__ == "__main__":
  main()
