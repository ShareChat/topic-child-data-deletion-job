### Topic Child Table deletion Job

This job takes care of data deletion of child tables of a topic. This is going to run only once

#### What does this job do ?

1. There is a temp topic table that is created. It reads that table in a paginated manner with query clause
```sql
SELECT topicId, lastUpdatedAt  FROM topic WHERE lastUpdatedAt < 1686375840000 AND (status IS NULL OR status='deleted' OR status='active') AND (giftingStatus IS NULL OR giftingStatus='DISABLED') LIMIT @limit OFFSET @offset
```
This is the exact same query on which the partitionedDML on actual topic table was deleted

2. It then puts the records returned to Pub/Sub - so that the consumer can delete take action on child tables

