@implementation Good

@property(weak) id<BadDelegate> delegate;
@property(weak, readonly) id<BadDelegate> delegate;
@property(weak, readwrite) id<BadDelegate> delegate;
@property(atomic, weak) id<BadDelegate> delegate;
@property(nonatomic, weak) id<BadDelegate> delegate;

@end
