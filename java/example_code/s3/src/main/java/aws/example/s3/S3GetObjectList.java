import software.amazon.awssdk.regions.Region;
import software.amazon.awssdk.services.s3.S3Client;
import software.amazon.awssdk.services.s3.model.ListObjectsV2Request;
import software.amazon.awssdk.services.s3.model.ListObjectsV2Response;
import software.amazon.awssdk.services.s3.model.S3Object;

public class S3GetObjectList {

    private static final String BUCKET_NAME = "your-bucket-name";

    public static void main(String[] args) {
        Region region = Region.US_EAST_1; // Change this to your desired AWS region

        S3Client s3Client = S3Client.builder().region(region).build();

        try {
            listObjects(s3Client);
        } catch (Exception e) {
            System.err.println("Error: " + e.getMessage());
        } finally {
            s3Client.close();
        }
    }

    public static void listObjects(S3Client s3Client) {
        ListObjectsV2Request listObjectsRequest = ListObjectsV2Request.builder()
                .bucket(BUCKET_NAME)
                .build();

        ListObjectsV2Response listObjectsResponse;
        do {
            listObjectsResponse = s3Client.listObjectsV2(listObjectsRequest);
            for (S3Object s3Object : listObjectsResponse.contents()) {
                System.out.println("Object key: " + s3Object.key());
                System.out.println("Object size: " + s3Object.size());
                // Add additional properties or actions as needed
            }
            listObjectsRequest = listObjectsRequest.toBuilder()
                    .continuationToken(listObjectsResponse.nextContinuationToken())
                    .build();
        } while (listObjectsResponse.isTruncated());
    }
}
