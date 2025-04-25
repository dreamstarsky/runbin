import base64
import gzip
import io

encoded_string = "H4sIAAAAAAAAAysODE7yi6oINipMzcwpdiszdYtKCjF2yknKdvOy9DIztIzMrDJOM/DJCkxx9igIDuYCAKWa4zUxAAAA"

try:
    # 1. Base64 Decode
    decoded_bytes = base64.b64decode(encoded_string)
    print(f"Base64 Decoded (bytes): {decoded_bytes[:20]}...") # Show first few bytes

    # 2. Gzip Decompress
    # Use BytesIO to treat the bytes like a file for gzip
    with io.BytesIO(decoded_bytes) as compressed_file:
        with gzip.GzipFile(fileobj=compressed_file, mode='rb') as decompressed_file:
            original_data = decompressed_file.read()

    # Decode the resulting bytes into a string (assuming UTF-8)
    original_text = original_data.decode('utf-8')

    print(f"\nDecoded and Decompressed Text: {original_text}")

except Exception as e:
    print(f"An error occurred: {e}")

