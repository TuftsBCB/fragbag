require 'rubygems'
require 'bio'
# require 'peach'

# ruby lib/coalesce_tree.rb /h/ndaniels/astral_true.tree /h/ndaniels/astral_data/astral_matt_output /h/ndaniels/cluster_assignments.csv
class Bio::Tree::Node

  @@thresholds = {:family => 0.095804, :superfamily => 0.0996, :fold => 0.102081}
  
  @@fold_members = {}
  @@superfamily_members = {}
  @@family_members = {}
  
  @@fold_counter = 0
  @@superfamily_counter = 0
  @@family_counter = 0
  
  # @@lock = Mutex.new
  
  def self.distance_matrix=(m)
    @@distance_matrix = m
  end
  
  def self.distance_matrix
    @@distance_matrix
  end
  
  def distance_matrix
    @@distance_matrix
  end
  
  def thresholds
    @@thresholds
  end
  
  def self.tree=(t)
    @@tree = t
  end
  
  def tree
    @@tree
  end
  
  def descendents
    tree.descendents(self)
  end
  
  def children
    tree.children(self)
  end
  
  def leaves
    tree.leaves(self)
  end
  
  def path(other)
    tree.path(self,other)
  end
  
  def depth
    tree.path(tree.root, self).length - 1
  end
  
  def parent
    tree.parent(self)
  end
  
  def siblings
    self.parent.children.find_all{|e| e != self}
  end
  
  def leaf?
    self.children.empty?
  end
  
  def collapsible(candidates)
    # candidates should be a hash like close_enough
    # every leaf must be within threshold of every other
    
    
    # TODO have a threshold counter... we bail when bad pairs is finally > (1-threshold)*length
    
    
    
    close_enough = {:family => true, :superfamily => true, :fold => true}
    if ! self.leaf?
      my_leaves = self.leaves # this is for speed
      catch :short_circuit do
        my_leaves.each_with_index do |ks1, i1| 
          k1 = ks1.name.gsub(' ','_')
          k1sym = k1.to_sym
          my_leaves.each_with_index do |ks2, i2| 
            next unless i2 > i1
            k2=ks2.name.gsub(' ','_') # seems expensive
            k2sym = k2.to_sym
            
            
            # if ! @@distance_matrix[k1] && ! @@distance_matrix[k2] # going to involve checking NULL on a hash lookup
            # 
            #   puts "Uh oh! k1: #{k1} k2: #{k2}"
            #   puts "Distance matrix is a #{@@distance_matrix.class}"
            # end
            distance = (k1 < k2 ? @@distance_matrix[k1sym][k2sym] : @@distance_matrix[k2sym][k1sym])
            # puts "distance: #{distance}"
            if candidates[:family] && close_enough[:family] && distance <= thresholds[:family] # short_circuit these evals too
              # nop
              # puts "all"
            elsif candidates[:superfamily] && close_enough[:superfamily] && distance <= thresholds[:superfamily]
              close_enough[:family] = false
              # puts "not fam"
            elsif candidates[:fold] && close_enough[:fold] && distance <= thresholds[:fold]
              close_enough[:family] = false
              close_enough[:superfamily] = false
              # puts "not sf"
            else
              close_enough[:family] = false #### just set flags, and short circuit
              close_enough[:superfamily] = false
              close_enough[:fold] = false
              # puts "not fold"
              throw :short_circuit
            end # if
          end # inner loop
        end # outer loop
      end # catch
    end # if ! self.leaf?
    close_enough # return the hash of booleans
  end
  
  def display
    s = ""
    l = self.leaves
    if l.empty?
      s += "singleton: " + self.name
    else
      self.leaves.each do |l1|
        k1 = l1.name.gsub(' ','_')
        k1sym = k1.to_sym
        s += "#{l1.name}::  "
        s += self.leaves.map do |l2|
          k2 = l2.name.gsub(' ','_')
          k2sym = k2.to_sym
          distance = (k1 < k2 ? @@distance_matrix[k1sym][k2sym] : @@distance_matrix[k2sym][k1sym])
          "#{l2.name}: #{distance}"
        end.join(", ")
        s += "\n"
      end
    end
    s
  end
  
  def collapse(level)
    
    # consider the level
    # we're going to take all the leaves and put them into a new 'thing' of that level
    contents = case
    when self.leaf?
      [self.name.gsub(' ','_')]
    else
      leaves.collect{|l| l.name.gsub(' ','_')}
    end
    # @@lock.synchronize do
      case level
      when :family
        # STDERR.puts "Collapsing at #{level}. Counter is #{@@family_counter}"
        @@family_counter += 1
        @@family_members[@@family_counter] = contents
      when :superfamily
        # STDERR.puts "Collapsing at #{level}. Counter is #{@@superfamily_counter}"
        @@superfamily_counter += 1
        @@superfamily_members[@@superfamily_counter] = contents
      when :fold
        # STDERR.puts "Collapsing at #{level}. Counter is #{@@fold_counter}"
        @@fold_counter += 1
        @@fold_members[@@fold_counter] = contents
      end
    # end
  end
  
  def output_statistics
    puts "Family counter: #{@@family_counter}. Families: #{@@family_members.keys.length}"
    puts "Superfamily counter: #{@@superfamily_counter}. Superfamilies: #{@@superfamily_members.keys.length}"
    puts "Fold counter: #{@@fold_counter}. Folds: #{@@fold_members.keys.length}"
  end
  
  def handle_subtree(candidates)
    # use @@tree
    # handle the tree rooted at this node: do we collapse it in any way?
    # 
    d = self.depth
    if d < 2
      collapsible_hash = {:family => false, :superfamily => false, :fold => false}
    else
      collapsible_hash = self.collapsible(candidates)
    end
    raise "collapsible returned nil for #{self.inspect}" if collapsible_hash.nil?
    # we need to look at candidates each time, and see if we crossed 2 thresholds at once!
    
    
    # see if we collapse this subtree
    if collapsible_hash[:family]
      # base case!
      # collapse family
      self.collapse(:family)
      if candidates[:superfamily]
        self.collapse(:superfamily) 
        # puts "collapsing superfamily same time as family"
      end
      if candidates[:fold]
        self.collapse(:fold)
        # puts "collapsing fold same time as family"
        # puts "We have #{self.leaves.length} leaves"
        # puts self.display
        # puts "parent:"
        # puts self.parent.display
      end
      
      
      
    elsif collapsible_hash[:superfamily]
      # recurse on self.children
        # collapse superfamily
        self.collapse(:superfamily) if candidates[:superfamily]
        if candidates[:fold]
          self.collapse(:fold) 
          # puts "collapsing fold same time as superfamily"
        end
        
        
        # puts "recursing on family only"
        # and recurse on family
        children.each do |c| 
          c.handle_subtree({:family => candidates[:family], :superfamily => false, :fold => false})
        end
        
    elsif collapsible_hash[:fold]
      # collapse fold
      self.collapse(:fold) if candidates[:fold]
      
      # puts "recursing on family and superfamily"
      # and recurse on superfamily, family
      children.each do |c|
        c.handle_subtree({:family => candidates[:family], :superfamily => candidates[:superfamily], :fold => false})
      end
      
    else
      
      # recurse on fold, superfamily, family
      

      children.each do |c|
        c.handle_subtree({:family => candidates[:family], :superfamily => candidates[:superfamily], :fold => candidates[:fold]})
      end
      
    end
      
    # no need to actually return anything, right? perhaps return nil
    
    nil
  end
  
  def output_assignments(filename)

    folds = {}
    superfamilies = {}
    families = {}
    @@fold_members.each_pair do |k,v|
      v.each do |ve|
        folds[ve] = k
      end
    end
    @@superfamily_members.each_pair do |k,v|
      v.each do |ve|
        superfamilies[ve] = k
      end
    end
    @@family_members.each_pair do |k,v|
      v.each do |ve|
        families[ve] = k
      end
    end
    
    #now go through them all
    File.open(filename, 'w') do |f|
      folds.keys.each do |protein_name|
        f.puts [protein_name, families[protein_name].to_s, superfamilies[protein_name].to_s, folds[protein_name].to_s].join(",")
      end
    end
    
  end
  
end


tree_file = ARGV[0]
dirname = ARGV[1]
outfile = ARGV[2]

results = {}
fnum = 0
print "Reading files... "
STDOUT.flush
all_keys = {}
Dir.foreach(dirname) do |file|
  filename = File.join(dirname, file)
  if File.file?(filename)
    fnum += 1
    print "#{fnum} "
    STDOUT.flush
    File.foreach(filename) do |line|
      line.chomp!   
      # d1sp8a2.ent_d1l9xa_.ent:	38	2.536	0.4046	294.02	239.85	232.14	216	288
      identifier, corelen, rmsd, pval, s1, s2, s3, l1, l2 = line.split(/\t+/)
      core_value = (2*corelen.to_f / (l1.to_f + l2.to_f))
      if identifier =~ /^(\w+)\.ent_(\w+)\.ent:/
        left = Regexp.last_match[1]
        right = Regexp.last_match[2]

        dist = 1/(-6.04979701 * (rmsd.to_f - core_value*corelen.to_f * 0.155 + 1.6018) + 1000)*100
        
        # if dist < 0
        #   puts "line #{line}"
        #   puts "core_value #{core_value} core_len #{corelen} rmsd #{rmsd}"
        #   raise "Distance #{dist} for #{left} #{right} in file #{filename}"
        # end
        
        # instead of left & right, do them in sorted order
        k1, k2 = [left, right].sort.collect{|k| k.to_sym}
        all_keys[k1] = true
        all_keys[k2] = true
        results[k1] ||= {}
        # results[right] ||= {}
        results[k1][k2] = dist
        # results[right][left] = dist
      elsif line =~ /^Done.|^$/
        next
      else
        raise "Bad identifier #{identifier}"
      end
    end
  end
end
puts " Done."
# pull Newick tree from infile
print "Reading tree..."
instr = ''
File.foreach(tree_file) do |line|
  instr << line
end

tree = Bio::Newick.new(instr).tree

Bio::Tree::Node.tree = tree

Bio::Tree::Node.distance_matrix = results

root = tree.root
puts " Done."
print "Merging tree..."
root.handle_subtree({:family => true, :superfamily => true, :fold => true})
puts " Done."
root.output_statistics
print "Outputting assignments..."
root.output_assignments(outfile)
puts " Done."
# we want to recursively descend the tree, collapsing subtrees that are within each threshold

# so we call collapsible on each subtree node, and if we can collapse it at the family level we don't recurse further

# if we collapse it at a higher level, we pass that down

# want to be able to do the distance-testing in parallel

# now how to create and merge clusters? ids for each distinct level.

# have an id pool for each.

# heuristic - need an overall depth for the tree; no need at all to do comparisons on things with depth < 3

# recursive or iterative? recursive but keep the tree global


