import software.amazon.awssdk.regions.Region;
import software.amazon.awssdk.services.s3.S3Client;
import software.amazon.awssdk.services.s3.model.CopyObjectRequest;
import software.amazon.awssdk.services.s3.model.ListObjectsV2Request;
import software.amazon.awssdk.services.s3.model.ListObjectsV2Response;
import software.amazon.awssdk.services.s3.model.S3Object;

public class S3CopyFiles {

    private static final String SOURCE_BUCKET_NAME = "source-bucket-name";
    private static final String DESTINATION_BUCKET_NAME = "destination-bucket-name";

    public static void main(String[] args) {
        Region region = Region.US_EAST_1; // Change this to your desired AWS region

        S3Client s3Client = S3Client.builder().region(region).build();

        try {
            copyObjects(s3Client);
        } catch (Exception e) {
            System.err.println("Error: " + e.getMessage());
        } finally {
            s3Client.close();
        }
    }

    public static void copyObjects(S3Client s3Client) {
        ListObjectsV2Request listObjectsRequest = ListObjectsV2Request.builder()
                .bucket(SOURCE_BUCKET_NAME)
                .build();

        ListObjectsV2Response listObjectsResponse;
        do {
            listObjectsResponse = s3Client.listObjectsV2(listObjectsRequest);
            for (S3Object s3Object : listObjectsResponse.contents()) {
                String key = s3Object.key();
                System.out.println("Copying object: " + key);
                copyObject(s3Client, key);
            }
            listObjectsRequest = listObjectsRequest.toBuilder()
                    .continuationToken(listObjectsResponse.nextContinuationToken())
                    .build();
        } while (listObjectsResponse.isTruncated());
    }

    public static void copyObject(S3Client s3Client, String key) {
        CopyObjectRequest copyObjectRequest = CopyObjectRequest.builder()
                .sourceBucket(SOURCE_BUCKET_NAME)
                .sourceKey(key)
                .destinationBucket(DESTINATION_BUCKET_NAME)
                .destinationKey(key) // You can specify a different key if you want
                .build();
        s3Client.copyObject(copyObjectRequest);
        System.out.println("Copied object: " + key);
    }
}
