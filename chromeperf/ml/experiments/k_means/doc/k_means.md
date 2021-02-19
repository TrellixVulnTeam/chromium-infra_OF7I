# The approach
We approach the data with the aim of understanding the underlying distribution
of the sample values for each label within a metric. To do so, we wish to
understand where natural clusters form within the data in order to identify
potential shifts in the distribution of data over time. We decided to employ the
use of k-means clustering because

1. It is relatively straightforward to implement
1. It scales well with large datasets
1. It can deal with clusters of varying shapes and sizes
1. Provides clear visual indicators of movement


# K-means Clustering
The K-means clustering approach takes unlabelled data and splits it into a set
number of (k) clusters. For each label dataset we will find the optimal number
of clusters by using the elbow and silhoutte methods.
## Elbow method
This is a heuristic used in determining the number of clusters in a data set.
The method consists of plotting the explained variation as a function of the
number of clusters, and picking the elbow of the curve as the number of clusters
to use. The elbow is the cutoff point of the curve after which any increase in
clusters doesn't make a significant impact. The idea is that the first few
clusters provide the bulk of the information (explain a lot of variation), since
the data naturally forms these set groups (so these clusters are necessary), but
once the number of clusters exceeds the natural groupings, the added information
will drop sharply, because it is just subdividing the actual groups. At this
point there will appear a sharp elbow in the graph and this point returns the
optimal k value for our k-means clustering.

## Silhoutte method
We use the Silhoutte score to find out the quality of clustering for a set of
data. It is used to measure how dense and well-separated the clusters are. The
value of the Silhouette score varies from -1 to 1. 1 indicates dense clustering
with well separated clusters. 0 indicates overlapping clusters with unclear
cluster distinction and samples very close to the decision boundary of the
neighboring clusters. A negative score suggests the data may have been assigned
to incorrect clusters.

For this approach, we find the silhoutte score for a range of different k
values. Our aim is to find which k value (number of clusters) results in the
highest silhoutte score. This then becomes the k value that we use in the
k-means clustering.

Literature claims the silhoutte method to be more reliable than the Elbow method
so we rely on the value from the silhoutte method to inform our k value
decision.


# Sliding windows
To increase our sample size and reduce the effects of noise at an individual
revision, we create sliding windows. A window spans 50 revisions. We agregate
all the sample values within each window. Then we can track how the centroid of
each cluster shifts over multiple revision windows.
