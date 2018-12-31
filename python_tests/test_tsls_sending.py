import hashlib
import socket
import array
import struct
from unittest import TestCase
from uuid import uuid4

import settings

SIGNATURE_SIZE = 1024 * 8
SIGNATURES_COUNT = 12

DATA_TYPE_HEADER = 64


class TSLSendingTest(TestCase):
    @staticmethod
    def generate_tsl():
        uuid = array.array('B', uuid4().bytes)
        signatures = array.array('B', [0] * SIGNATURE_SIZE * SIGNATURES_COUNT)

        data = uuid + array.array('B', struct.pack("H", SIGNATURES_COUNT)) + signatures
        h = hashlib.blake2b(digest_size=32)
        h.update(data)

        return data + array.array('B', h.digest())

    def test_sending(self):
        s = socket.socket(socket.AF_INET, socket.SOCK_STREAM)
        s.connect((settings.OBSERVER_HOST, settings.OBSERVER_PORT))

        message = self.generate_tsl()
        s.send(struct.pack("LB", len(message), DATA_TYPE_HEADER))
        s.send(message)

        response = s.recv(1)
        self.assertEqual(response, b'\x00')

        s.close()
