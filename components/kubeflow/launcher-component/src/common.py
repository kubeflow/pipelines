# -*- coding: utf-8 -*-
"""
@author: chongchuanbing
"""

from enum import Enum


class Status(Enum):
    Running = 0
    Failed = -1
    Succeed = 1
