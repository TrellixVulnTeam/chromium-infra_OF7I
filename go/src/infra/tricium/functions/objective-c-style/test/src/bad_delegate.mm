
@implementation Bad

@property(strong) id<BadDelegate> delegate;
@property(strong, readonly) id<BadDelegate> delegate;
@property(strong, readwrite) id<BadDelegate> delegate;
@property(atomic, strong) id<BadDelegate> delegate;
@property(nonatomic, strong) id<BadDelegate> delegate;
@property(atomic) id<BadDelegate> delegate;
@property(nonatomic) id<BadDelegate> delegate;
@property(readwrite) id<BadDelegate> delegate;
@property(readonly) id<BadDelegate> delegate;
@property(readonly) id<BadDelegate> superDelegate;

@end
