#!/usr/bin/env python3
"""
Manual decryption script for aws-s3-backup encrypted files
Usage: python3 decrypt_manual.py encrypted_file.enc password
"""

import sys
import os
from cryptography.hazmat.primitives.kdf.scrypt import Scrypt
from cryptography.hazmat.primitives.ciphers.aead import AESGCM
from cryptography.hazmat.backends import default_backend

def decrypt_file(encrypted_file, password):
    """Decrypt a file encrypted by aws-s3-backup"""
    
    # Read encrypted data
    with open(encrypted_file, 'rb') as f:
        data = f.read()
    
    # Minimum size check
    if len(data) < 32 + 12 + 16:  # salt(32) + nonce(12) + tag(16)
        raise ValueError("File too short to be valid encrypted data")
    
    # Extract salt (last 32 bytes) and ciphertext
    salt = data[-32:]
    ciphertext_with_nonce = data[:-32]
    
    # Try new parameters first (N=131072)
    try:
        return try_decrypt_with_params(ciphertext_with_nonce, password.encode(), salt, 131072)
    except:
        pass
    
    # Try legacy parameters (N=32768)
    try:
        return try_decrypt_with_params(ciphertext_with_nonce, password.encode(), salt, 32768)
    except:
        pass
    
    raise ValueError("Decryption failed with both parameter sets")

def try_decrypt_with_params(ciphertext_with_nonce, password, salt, N):
    """Try decryption with specific scrypt parameters"""
    
    # Derive key using scrypt
    kdf = Scrypt(
        algorithm=None,
        length=32,
        salt=salt,
        n=N,
        r=8,
        p=1,  # Simplified for manual decryption
        backend=default_backend()
    )
    key = kdf.derive(password)
    
    # Extract nonce (first 12 bytes) and ciphertext
    nonce = ciphertext_with_nonce[:12]
    ciphertext = ciphertext_with_nonce[12:]
    
    # Decrypt using AES-GCM
    aesgcm = AESGCM(key)
    plaintext = aesgcm.decrypt(nonce, ciphertext, None)
    
    return plaintext

def main():
    if len(sys.argv) != 3:
        print("Usage: python3 decrypt_manual.py encrypted_file.enc password")
        sys.exit(1)
    
    encrypted_file = sys.argv[1]
    password = sys.argv[2]
    
    if not os.path.exists(encrypted_file):
        print(f"Error: File {encrypted_file} not found")
        sys.exit(1)
    
    try:
        # Decrypt the file
        decrypted_data = decrypt_file(encrypted_file, password)
        
        # Write decrypted data
        output_file = encrypted_file.replace('.enc', '')
        with open(output_file, 'wb') as f:
            f.write(decrypted_data)
        
        print(f"✅ Successfully decrypted: {output_file}")
        
    except Exception as e:
        print(f"❌ Decryption failed: {e}")
        sys.exit(1)

if __name__ == "__main__":
    main()