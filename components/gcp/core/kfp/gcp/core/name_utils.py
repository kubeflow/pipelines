
import logging
import re

def normalize(name,
              allowed_char_pattern='0-9a-zA-Z',
              replace_text='_',
              first_char_pattern='[a-zA-Z]',
              prefix_text='x_'):
        if not name:
            return name
        normalized_name = re.sub('[^{}]+'.format(allowed_char_pattern), replace_text, name)
        if not re.match(first_char_pattern, normalized_name[0]):
            normalized_name = prefix_text + normalized_name
        if name != normalized_name:
            logging.info('Normalize name from "{}" to "{}".'.format(name, normalized_name))
        return normalized_name