The following command grants ``full control`` to two AWS users (*user1@example.com* and *user2@example.com*) and ``read``
permission to everyone::

   aws s3api put-object-acl --bucket amzn-s3-demo-bucket --key file.txt --grant-full-control emailaddress=user1@example.com,emailaddress=user2@example.com --grant-read uri=http://acs.amazonaws.com/groups/global/AllUsers

See http://docs.aws.amazon.com/AmazonS3/latest/API/RESTBucketPUTacl.html for details on custom ACLs (the s3api ACL
commands, such as ``put-object-acl``, use the same shorthand argument notation).
