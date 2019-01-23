import socket
import array
import struct
from unittest import TestCase
from uuid import uuid4

import settings


MEMBER_SIZE = (1024 * 8) + 2
MEMBERS_COUNT = 3


class TSLAttemptTest(TestCase):
    @staticmethod
    def generate_tsl():
        uuid = array.array('B', uuid4().bytes)
        members = array.array('B', [0] * MEMBER_SIZE * MEMBERS_COUNT)
        return uuid + array.array('B', struct.pack("H", MEMBERS_COUNT)) + members

    def setUp(self):
        self.sock = socket.socket(socket.AF_INET, socket.SOCK_STREAM)
        self.sock.connect((settings.OBSERVER_HOST, settings.OBSERVER_PORT))

    def tearDown(self):
        self.sock.close()

    def send_tsl(self, message):
        self.sock.send(struct.pack("L", len(message) + 2))
        self.sock.send(struct.pack("B", 0))
        self.sock.send(struct.pack("B", 64))
        self.sock.send(message)

    def test_valid_tsl(self):
        self.send_tsl(self.generate_tsl())

        # todo: check pool
