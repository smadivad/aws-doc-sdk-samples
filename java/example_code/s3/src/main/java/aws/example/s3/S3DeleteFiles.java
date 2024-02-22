import software.amazon.awssdk.regions.Region;
import software.amazon.awssdk.services.s3.S3Client;
import software.amazon.awssdk.services.s3.model.DeleteObjectRequest;
import software.amazon.awssdk.services.s3.model.ListObjectsV2Request;
import software.amazon.awssdk.services.s3.model.ListObjectsV2Response;
import software.amazon.awssdk.services.s3.model.S3Object;

public class S3DeleteFiles {

    private static final String BUCKET_NAME = "your-bucket-name";

    public static void main(String[] args) {
        Region region = Region.US_EAST_1; // Change this to your desired AWS region

        S3Client s3Client = S3Client.builder().region(region).build();

        try {
            listAndDeleteObjects(s3Client);
        } catch (Exception e) {
            System.err.println("Error: " + e.getMessage());
        } finally {
            s3Client.close();
        }
    }

    public static void listAndDeleteObjects(S3Client s3Client) {
        ListObjectsV2Request listObjectsRequest = ListObjectsV2Request.builder()
                .bucket(BUCKET_NAME)
                .build();

        ListObjectsV2Response listObjectsResponse;
        do {
            listObjectsResponse = s3Client.listObjectsV2(listObjectsRequest);
            for (S3Object s3Object : listObjectsResponse.contents()) {
                String key = s3Object.key();
                System.out.println("Deleting object: " + key);
                deleteObject(s3Client, key);
            }
            listObjectsRequest = listObjectsRequest.toBuilder()
                    .continuationToken(listObjectsResponse.nextContinuationToken())
                    .build();
        } while (listObjectsResponse.isTruncated());
    }

    public static void deleteObject(S3Client s3Client, String key) {
        DeleteObjectRequest deleteObjectRequest = DeleteObjectRequest.builder()
                .bucket(BUCKET_NAME)
                .key(key)
                .build();
        s3Client.deleteObject(deleteObjectRequest);
        System.out.println("Deleted object: " + key);
    }
}
