import fire
import logging

from ml_engine.ml_engine import MLEngine

def main():
    logging.basicConfig(level=logging.INFO)
    fire.Fire(MLEngine, name='mlengine')

if __name__ == '__main__':
    main()