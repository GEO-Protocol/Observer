import hashlib
import socket
import array
import struct
from unittest import TestCase
from uuid import uuid4

import settings

PUB_KEY_SIZE = 1024 * 16
PUB_KEYS_COUNT = 12

DATA_TYPE_HEADER = 65


class ClaimsSendingTest(TestCase):
    @staticmethod
    def generate_claim():
        uuid = array.array('B', uuid4().bytes)
        pub_keys = array.array('B', [0] * PUB_KEY_SIZE * PUB_KEYS_COUNT)

        data = uuid + array.array('B', struct.pack("H", PUB_KEYS_COUNT)) + pub_keys
        h = hashlib.blake2b(digest_size=32)
        h.update(data)

        return data + array.array('B', h.digest())

    def test_sending(self):
        s = socket.socket(socket.AF_INET, socket.SOCK_STREAM)
        s.connect((settings.OBSERVER_HOST, settings.OBSERVER_PORT))

        message = self.generate_claim()
        s.send(struct.pack("LB", len(message), DATA_TYPE_HEADER))
        s.send(message)

        response = s.recv(1)
        self.assertEqual(response, b'\x00')

        s.close()
