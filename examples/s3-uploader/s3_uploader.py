#!/usr/bin/env python3
"""
S3 Uploader Example for Local Development
This script demonstrates how to use AWS services with LocalStack
"""

import boto3
import os
from datetime import datetime
import json

class LocalStackS3Uploader:
    def __init__(self, use_localstack=True):
        """Initialize the S3 uploader with LocalStack or AWS"""
        if use_localstack:
            # LocalStack configuration
            self.s3_client = boto3.client(
                's3',
                endpoint_url='http://localhost:4566',
                aws_access_key_id='test',
                aws_secret_access_key='test',
                region_name='eu-central-1'
            )
            self.bucket_name = 'development-bucket'
        else:
            # Real AWS configuration (for playground)
            self.s3_client = boto3.client('s3', region_name='eu-central-1')
            self.bucket_name = 'aws-playground-bucket'
        
        self.use_localstack = use_localstack
    
    def create_bucket_if_not_exists(self):
        """Create bucket if it doesn't exist"""
        try:
            self.s3_client.head_bucket(Bucket=self.bucket_name)
            print(f"Bucket '{self.bucket_name}' already exists")
        except:
            print(f"Creating bucket '{self.bucket_name}'...")
            self.s3_client.create_bucket(
                Bucket=self.bucket_name,
                CreateBucketConfiguration={
                    'LocationConstraint': 'eu-central-1'
                }
            )
            print(f"Bucket '{self.bucket_name}' created successfully")
    
    def upload_file(self, file_path, object_name=None):
        """Upload a file to S3"""
        if not os.path.exists(file_path):
            raise FileNotFoundError(f"File not found: {file_path}")
        
        if object_name is None:
            object_name = os.path.basename(file_path)
        
        try:
            self.s3_client.upload_file(file_path, self.bucket_name, object_name)
            print(f"Uploaded '{file_path}' to '{self.bucket_name}/{object_name}'")
            
            # Generate presigned URL (works with LocalStack too)
            url = self.s3_client.generate_presigned_url(
                'get_object',
                Params={'Bucket': self.bucket_name, 'Key': object_name},
                ExpiresIn=3600
            )
            print(f"Presigned URL (valid for 1 hour): {url}")
            
            return object_name
        except Exception as e:
            print(f"Error uploading file: {e}")
            raise
    
    def list_objects(self):
        """List all objects in the bucket"""
        try:
            response = self.s3_client.list_objects_v2(Bucket=self.bucket_name)
            
            if 'Contents' in response:
                print(f"\nObjects in bucket '{self.bucket_name}':")
                for obj in response['Contents']:
                    print(f"  - {obj['Key']} ({obj['Size']} bytes, last modified: {obj['LastModified']})")
            else:
                print(f"No objects found in bucket '{self.bucket_name}'")
            
            return response.get('Contents', [])
        except Exception as e:
            print(f"Error listing objects: {e}")
            return []
    
    def download_file(self, object_name, download_path):
        """Download a file from S3"""
        try:
            self.s3_client.download_file(self.bucket_name, object_name, download_path)
            print(f"Downloaded '{object_name}' to '{download_path}'")
            return download_path
        except Exception as e:
            print(f"Error downloading file: {e}")
            raise
    
    def create_test_data(self):
        """Create test data and upload it"""
        test_data = {
            "timestamp": datetime.now().isoformat(),
            "environment": "localstack" if self.use_localstack else "aws-playground",
            "data": {
                "sample_key": "sample_value",
                "numbers": [1, 2, 3, 4, 5],
                "nested": {
                    "field": "value"
                }
            }
        }
        
        # Create test file
        test_file = "test_data.json"
        with open(test_file, 'w') as f:
            json.dump(test_data, f, indent=2)
        
        print(f"Created test file: {test_file}")
        return test_file

def main():
    """Main function to demonstrate S3 operations"""
    print("=" * 60)
    print("AWS LocalStack S3 Uploader Demo")
    print("=" * 60)
    
    # Initialize with LocalStack
    uploader = LocalStackS3Uploader(use_localstack=True)
    
    try:
        # Step 1: Create bucket
        uploader.create_bucket_if_not_exists()
        
        # Step 2: Create and upload test data
        print("\n" + "=" * 60)
        print("Step 1: Creating and uploading test data")
        print("=" * 60)
        
        test_file = uploader.create_test_data()
        uploaded_key = uploader.upload_file(test_file, "test-data.json")
        
        # Step 3: List objects
        print("\n" + "=" * 60)
        print("Step 2: Listing bucket contents")
        print("=" * 60)
        
        uploader.list_objects()
        
        # Step 4: Download file
        print("\n" + "=" * 60)
        print("Step 3: Downloading file")
        print("=" * 60)
        
        download_path = "downloaded_test_data.json"
        uploader.download_file(uploaded_key, download_path)
        
        # Verify download
        with open(download_path, 'r') as f:
            downloaded_data = json.load(f)
            print(f"\nDownloaded data verification:")
            print(f"  Timestamp: {downloaded_data['timestamp']}")
            print(f"  Environment: {downloaded_data['environment']}")
        
        # Cleanup
        os.remove(test_file)
        os.remove(download_path)
        print(f"\nCleaned up temporary files")
        
        print("\n" + "=" * 60)
        print("Demo completed successfully!")
        print("=" * 60)
        
    except Exception as e:
        print(f"\nError during demo: {e}")
        print("Make sure LocalStack is running: task localstack:up")
        return 1
    
    return 0

if __name__ == "__main__":
    exit(main())
