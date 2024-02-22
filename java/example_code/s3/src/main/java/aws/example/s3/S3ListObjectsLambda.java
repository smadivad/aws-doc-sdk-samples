import com.amazonaws.services.lambda.runtime.Context;
import com.amazonaws.services.lambda.runtime.RequestHandler;
import software.amazon.awssdk.regions.Region;
import software.amazon.awssdk.services.s3.S3Client;
import software.amazon.awssdk.services.s3.model.ListObjectsRequest;
import software.amazon.awssdk.services.s3.model.ListObjectsResponse;
import software.amazon.awssdk.services.s3.model.S3Object;

import java.util.List;

public class S3ListObjectsLambda implements RequestHandler<String, String> {

    private static final String BUCKET_NAME = "your-bucket-name";

    @Override
    public String handleRequest(String input, Context context) {
        Region region = Region.US_EAST_1; // Change this to your desired AWS region

        try {
            S3Client s3Client = S3Client.builder().region(region).build();

            ListObjectsRequest listObjectsRequest = ListObjectsRequest.builder()
                    .bucket(BUCKET_NAME)
                    .build();

            ListObjectsResponse listObjectsResponse = s3Client.listObjects(listObjectsRequest);
            List<S3Object> objects = listObjectsResponse.contents();

            StringBuilder response = new StringBuilder();
            response.append("Objects in bucket ").append(BUCKET_NAME).append(":\n");

            for (S3Object object : objects) {
                response.append(object.key()).append("\n");
            }

            return response.toString();
        } catch (Exception e) {
            return "Error: " + e.getMessage();
        }
    }
}
