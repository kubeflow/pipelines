import fire
import logging

from dataflow.dataflow import Dataflow

def main():
    logging.basicConfig(level=logging.INFO)
    fire.Fire(Dataflow, name='dataflow')

if __name__ == '__main__':
    main()