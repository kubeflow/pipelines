import argparse
import sys
import toml

parser = argparse.ArgumentParser(
    description='Parse toml format go dependencies maintained by dep tool.')
parser.add_argument('dep_lock_path',
                    nargs='?',
                    default='Gopkg.lock',
                    help='Toml format go dependency lock file.')
parser.add_argument(
    '-o',
    '--output',
    dest='output_file',
    nargs='?',
    default='dep.txt',
    help='Output file with one line per golang library. (default: %(default)s)',
)

args = parser.parse_args()


def main():
    print('Parsing dependencies from {}'.format(args.dep_lock_path),
          file=sys.stderr)

    with open(args.output_file, 'w') as output_file:
        deps = toml.load(args.dep_lock_path)
        projects = deps.get('projects')
        dep_names = list(map(lambda p: p.get('name'), projects))
        for name in dep_names:
            print(name, file=output_file)

        print('Found {} dependencies'.format(len(projects)), file=sys.stderr)


if __name__ == '__main__':
    main()
