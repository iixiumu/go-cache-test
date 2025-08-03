# go-cache

实现go语言高级缓存库，Cacher是业务层缓存抽象，内部使用Store作为存储后端，提供更高级的缓存模式和回退策略

Store和Cacher接口定义已经有了，按以下步骤实现：

- 为Cacher接口设计单元测试
- 实现Cacher接口
- 基于redis/go-redis设计单元测试，然后实现Store接口
- hypermodeinc/ristretto设计单元测试，然后实现Store接口
- 基于bluele/gcache设计单元测试，然后实现Store接口

注意以下几点：

- redis单元测试可以用alicebob/miniredis
- 由于缓存库要支持多种数据类型，用泛型使用很麻烦，所以就用反射实现
- 对于内存缓存，可以直接存储对象，所以不要用序列化和反序列化
- 对于redis，序列化和反序列化用json即可
- 各种库的使用方法，可以使用context7
