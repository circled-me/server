from PIL import Image, ImageDraw
import face_recognition
import sys
import json
import numpy as np

class NumpyEncoder(json.JSONEncoder):
    def default(self, obj):
        if isinstance(obj, np.ndarray):
            return obj.tolist()
        return super().default(obj)

# Read a line from the standard input
while True:
    line = sys.stdin.readline().strip()
    if not line or line == '':
        break
    if line == 'ping':
        print('pong')
        sys.stdout.flush()
        continue

    # Load the image from the line
    image = face_recognition.load_image_file(line)
    # Find all face locations in the image
    face_locations = face_recognition.face_locations(image, model='hog')
    # Extract all face encodings
    face_encodings = face_recognition.face_encodings(image, face_locations, model='large')

    # Return a JSON object with the face locations and encodings
    print(
        json.dumps({
            'locations': face_locations,
            'encodings': face_encodings
        }, 
        cls=NumpyEncoder)
    )
    sys.stdout.flush()
